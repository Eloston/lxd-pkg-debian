/*
** 2018 February 19
**
** The author disclaims copyright to this source code.  In place of
** a legal notice, here is a blessing:
**
**    May you do good and not evil.
**    May you find forgiveness for yourself and forgive others.
**    May you share freely, never taking more than you give.
**
*************************************************************************
**
** This file contains code used for testing the SQLite system.
** None of the code in this file goes into a deliverable build.
**
** This file contains a stub implementation of the write-ahead log replication
** interface. It can be used by tests to exercise the WAL replication APIs
** exposed by SQLite.
**
** This replication implementation is designed for testability and does
** not involve any actual networking.
*/
#if defined(SQLITE_ENABLE_WAL_REPLICATION) && !defined(SQLITE_OMIT_WAL)

#if defined(INCLUDE_SQLITE_TCL_H)
#  include "sqlite_tcl.h"
#else
#  include "tcl.h"
#endif

#include "sqliteInt.h"
#include "sqlite3.h"
#include <assert.h>

extern const char *sqlite3ErrName(int);

/* These functions are implemented in test1.c. */
extern int getDbPointer(Tcl_Interp *, const char *, sqlite3 **);

/* Hold information about a single WAL frame that was passed to the
** sqlite3_wal_replication.xFrames method implemented in this file.
**
** This is used for test assertions.
*/
typedef struct testWalReplicationFrameInfo testWalReplicationFrameInfo;
struct testWalReplicationFrameInfo {
  unsigned szPage;   /* Number of bytes in the frame's page */
  unsigned pgno;     /* Page number */
  unsigned iPrev;    /* Most recent frame also containing pgno, or 0 if new */

  /* Linked list of frame info objects maintained by testWalReplicationFrames,
  ** head is the newest and tail the oldest. */
  testWalReplicationFrameInfo* pNext;
};

/*
** Global WAL replication context used by this stub implementation of
** sqlite3_wal_replication_wal. It holds a state variable that captures the current
** WAL lifecycle phase and it optionally holds a pointer to a connection in
** follower WAL replication mode.
*/
typedef struct testWalReplicationContextType testWalReplicationContextType;
struct testWalReplicationContextType {
  int eState;          /* Replication state (IDLE, PENDING, WRITING, etc) */
  int eFailing;        /* Code of a method that should fail when triggered */
  int rc;              /* If non-zero, the eFailing method will error */
  int iFailures;       /* Number of times the eFailing method will error */
  sqlite3 *db;         /* Follower connection */
  const char *zSchema; /* Follower schema name */

  /* List of all frames that were passed to the xFrames hook since the last
  ** context reset.
  */
  testWalReplicationFrameInfo *pFrameList;
};
static testWalReplicationContextType testWalReplicationContext;

#define STATE_IDLE      0
#define STATE_PENDING   1
#define STATE_WRITING   2
#define STATE_COMMITTED 3
#define STATE_UNDONE    4
#define STATE_ERROR     5

#define FAILING_BEGIN  1
#define FAILING_FRAMES 2
#define FAILING_UNDO   3
#define FAILING_END    4

/* Reset the state of the global WAL replication context */
static void testWalReplicationContextReset() {
  testWalReplicationFrameInfo *pFrame;
  testWalReplicationFrameInfo *pFrameNext;

  testWalReplicationContext.eState = STATE_IDLE;
  testWalReplicationContext.eFailing = 0;
  testWalReplicationContext.rc = 0;
  testWalReplicationContext.iFailures = 8192; /* Effetively infinite */
  testWalReplicationContext.db = 0;
  testWalReplicationContext.zSchema = 0;

  /* Free all memory allocated for frame info objects */
  pFrame = testWalReplicationContext.pFrameList;
  while( pFrame ){
    pFrameNext = pFrame->pNext;
    sqlite3_free(pFrame);
    pFrame = pFrameNext;
  }

  testWalReplicationContext.pFrameList = 0;
}

/*
** A version of sqlite3_wal_replication.xBegin() that transitions the global
** replication context state to STATE_PENDING.
*/
static int testWalReplicationBegin(
  sqlite3_wal_replication *pReplication, void *pArg
){
  int rc = SQLITE_OK;
  assert( pArg==&testWalReplicationContext );
  assert( testWalReplicationContext.eState==STATE_IDLE
       || testWalReplicationContext.eState==STATE_ERROR
  );
  if( testWalReplicationContext.eFailing==FAILING_BEGIN
   && testWalReplicationContext.iFailures>0
  ){
    rc = testWalReplicationContext.rc;
    testWalReplicationContext.iFailures--;
  }
  if( rc==SQLITE_OK ){
    testWalReplicationContext.eState = STATE_PENDING;
  }
  return rc;
}

/*
** A version of sqlite3_wal_replication.xAbort() that transitions the global
** replication context state to STATE_IDLE.
*/
static int testWalReplicationAbort(
  sqlite3_wal_replication *pReplication, void *pArg
){
  assert( pArg==&testWalReplicationContext );
  assert( testWalReplicationContext.eState==STATE_PENDING );
  testWalReplicationContext.eState = STATE_IDLE;
  return 0;
}

/*
** A version of sqlite3_wal_replication.xFrames() that invokes
** sqlite3_wal_replication_frames() on the follower connection configured in the
** global test replication context (if present).
*/
static int testWalReplicationFrames(
  sqlite3_wal_replication *pReplication, void *pArg,
  int szPage, int nFrame, sqlite3_wal_replication_frame *aFrame,
  unsigned nTruncate, int isCommit
){
  int rc = SQLITE_OK;
  int isBegin = 1;
  int i;
  sqlite3_wal_replication_frame *pNext;
  testWalReplicationFrameInfo *pFrame;

  assert( pArg==&testWalReplicationContext );
  assert( testWalReplicationContext.eState==STATE_PENDING
       || testWalReplicationContext.eState==STATE_WRITING
  );

  /* Save information about these frames */
  pNext = aFrame;
  for (i=0; i<nFrame; i++) {
    pFrame = (testWalReplicationFrameInfo*)(sqlite3_malloc(
        sizeof(testWalReplicationFrameInfo)));
    if( !pFrame ){
	return SQLITE_NOMEM;
    }
    pFrame->szPage = szPage;
    pFrame->pgno = pNext->pgno;
    pFrame->iPrev = pNext->iPrev;
    pFrame->pNext = testWalReplicationContext.pFrameList;
    testWalReplicationContext.pFrameList = pFrame;
    pNext += 1;
  }

  if( testWalReplicationContext.eState==STATE_PENDING ){
    /* If the replication state is STATE_PENDING, it means that this is the
    ** first batch of frames of a new transaction. */
    isBegin = 1;
  }
  if( testWalReplicationContext.eFailing==FAILING_FRAMES
   && testWalReplicationContext.iFailures>0
  ){
    rc = testWalReplicationContext.rc;
    testWalReplicationContext.iFailures--;
  }else if( testWalReplicationContext.db ){
    unsigned *aPgno;
    void *aPage;
    int i;

    aPgno = sqlite3_malloc(sizeof(unsigned) * nFrame);
    if( !aPgno ){
      rc = SQLITE_NOMEM;
    }
    if( rc==SQLITE_OK ){
      aPage = (void*)sqlite3_malloc(sizeof(char) * szPage * nFrame);
    }
    if( !aPage ){
      sqlite3_free(aPgno);
      rc = SQLITE_NOMEM;
    }
    if( rc==SQLITE_OK ){
      for(i=0; i<nFrame; i++){
	aPgno[i] = aFrame[i].pgno;
	memcpy(aPage+(szPage*i), aFrame[i].pBuf, szPage);
      }
      rc = sqlite3_wal_replication_frames(
            testWalReplicationContext.db,
            testWalReplicationContext.zSchema,
            isBegin, szPage, nFrame, aPgno, aPage, nTruncate, isCommit
      );
      sqlite3_free(aPgno);
      sqlite3_free(aPage);
    }
  }
  if( rc==SQLITE_OK ){
    if( isCommit ){
      testWalReplicationContext.eState = STATE_COMMITTED;
    }else{
      testWalReplicationContext.eState = STATE_WRITING;
    }
  }else{
    testWalReplicationContext.eState = STATE_ERROR;
  }
  return rc;
}

/*
** A version of sqlite3_wal_replication.xUndo() that invokes
** sqlite3_wal_replication_undo() on the follower connection configured in the
** global test replication context (if present).
*/
static int testWalReplicationUndo(
  sqlite3_wal_replication *pReplication, void *pArg
){
  int rc = SQLITE_OK;
  assert( pArg==&testWalReplicationContext );
  assert( testWalReplicationContext.eState==STATE_PENDING
       || testWalReplicationContext.eState==STATE_WRITING
       || testWalReplicationContext.eState==STATE_ERROR
  );
  if( testWalReplicationContext.eFailing==FAILING_UNDO
   && testWalReplicationContext.iFailures>0
  ){
    rc = testWalReplicationContext.rc;
    testWalReplicationContext.iFailures--;
  }else if( testWalReplicationContext.db
         && testWalReplicationContext.eState==STATE_WRITING ){
    rc = sqlite3_wal_replication_undo(
        testWalReplicationContext.db,
        testWalReplicationContext.zSchema
    );
  }
  if( rc==SQLITE_OK ){
    testWalReplicationContext.eState = STATE_UNDONE;
  }
  return rc;
}

/*
** A version of sqlite3_wal_replication.xEnd() that transitions the global
** replication context state to STATE_IDLE.
*/
static int testWalReplicationEnd(
  sqlite3_wal_replication *pReplication, void *pArg
){
  int rc = SQLITE_OK;
  assert( pArg==&testWalReplicationContext );
  assert( testWalReplicationContext.eState==STATE_PENDING
       || testWalReplicationContext.eState==STATE_COMMITTED
       || testWalReplicationContext.eState==STATE_UNDONE
  );
  testWalReplicationContext.eState = STATE_IDLE;
  if( testWalReplicationContext.eFailing==FAILING_END
   && testWalReplicationContext.iFailures>0
  ){
    rc = testWalReplicationContext.rc;
    testWalReplicationContext.iFailures--;
  }
  return rc;
}

/*
** This function returns a pointer to the WAL replication implemented in this
** file.
*/
sqlite3_wal_replication *testWalReplication(void){
  static sqlite3_wal_replication replication = {
    1,
    0,
    "test",
    0,
    testWalReplicationBegin,
    testWalReplicationAbort,
    testWalReplicationFrames,
    testWalReplicationUndo,
    testWalReplicationEnd,
  };
  return &replication;
}

/*
** This function returns a pointer to the WAL replication implemented in this
** file, but using a different registration name than testWalRepl.
**
** It's used to exercise the WAL replication registration APIs.
*/
sqlite3_wal_replication *testWalReplicationAlt(void){
  static sqlite3_wal_replication replication = {
    1,
    0,
    "test-alt",
    0,
    testWalReplicationBegin,
    testWalReplicationAbort,
    testWalReplicationFrames,
    testWalReplicationUndo,
    testWalReplicationEnd,
  };
  return &replication;
}

/*
** tclcmd: sqlite3_wal_replication_find ?NAME?
**
** Return the name of the default WAL replication implementation, if one is
** registered, or no result otherwise.
**
** If NAME is passed, return NAME if a matching WAL replication implementation
** is registered, or no result otherwise.
*/
static int SQLITE_TCLAPI test_wal_replication_find(
  void * clientData,
  Tcl_Interp *interp,
  int objc,
  Tcl_Obj *CONST objv[]
){
  char *zName;
  sqlite3_wal_replication *pReplication;

  if( objc!=1 && objc!=2 ){
    Tcl_WrongNumArgs(interp, 2, objv, "?NAME?");
    return TCL_ERROR;
  }

  if( objc==2 ){
    zName = Tcl_GetString(objv[1]);
  }

  pReplication = sqlite3_wal_replication_find(zName);

  if( pReplication ){
    Tcl_AppendResult(interp, pReplication->zName, (char*)0);
  }

  return TCL_OK;
}

/*
** tclcmd: sqlite3_wal_replication_register DEFAULT ?ALT?
**
** Register the test write-ahead log replication implementation, with the name
** "test", making it the default if DEFAULT is 1.
**
** If the ALT flag is true, use "test-alt" as registration name.
*/
static int SQLITE_TCLAPI test_wal_replication_register(
  void * clientData,
  Tcl_Interp *interp,
  int objc,
  Tcl_Obj *CONST objv[]
){
  int bDefault = 0;
  int bAlt = 0;
  sqlite3_wal_replication *pReplication;

  if( objc!=2 && objc!=3 ){
    Tcl_WrongNumArgs(interp, 3, objv, "DEFAULT ?ALT?");
    return TCL_ERROR;
  }

  if( Tcl_GetIntFromObj(interp, objv[1], &bDefault) ){
    return TCL_ERROR;
  }

  if( objc==3 ){
    if( Tcl_GetIntFromObj(interp, objv[2], &bAlt) ){
      return TCL_ERROR;
    }
  }

  if( bAlt==0 ){
    pReplication = testWalReplication();
  }else{
    pReplication = testWalReplicationAlt();
  }

  sqlite3_wal_replication_register(pReplication, bDefault);

  return TCL_OK;
}

/*
** tclcmd: sqlite3_wal_replication_unregister ?ALT?
**
** Unregister the test write-ahead log replication implementation.
**
** If the ALT flag is true, unregister the alternate implementation.
*/
static int SQLITE_TCLAPI test_wal_replication_unregister(
  void * clientData,
  Tcl_Interp *interp,
  int objc,
  Tcl_Obj *CONST objv[]
){
  int bAlt = 0;

  if( objc!=1 && objc!=2 ){
    Tcl_WrongNumArgs(interp, 2, objv, "?ALT?");
    return TCL_ERROR;
  }

  if( objc==2 ){
    if( Tcl_GetIntFromObj(interp, objv[1], &bAlt) ){
      return TCL_ERROR;
    }
  }

  if( bAlt==0 ){
    sqlite3_wal_replication_unregister(testWalReplication());
  }else{
    sqlite3_wal_replication_unregister(testWalReplicationAlt());
  }
  return TCL_OK;
}

/*
** tclcmd: sqlite3_wal_replication_error METHOD ERROR ?N?
**
** Make the given method of test WAL replication implementation fail with the
** given error. If N is given, fail only that amount of time and start
** succeeding again afterwise.
*/
static int SQLITE_TCLAPI test_wal_replication_error(
  void * clientData,
  Tcl_Interp *interp,
  int objc,
  Tcl_Obj *CONST objv[]
){
  const char *zMethod;
  const char *zError;
  int eFailing;
  int rc;
  int iFailures;

  if( objc!=3 && objc!=4 ){
    Tcl_WrongNumArgs(interp, 3, objv, "METHOD ERROR ?N?");
    return TCL_ERROR;
  }

  /* Failing method */
  zMethod = Tcl_GetString(objv[1]);
  if( strcmp(zMethod, "xBegin")==0 ){
    eFailing = FAILING_BEGIN;
  }else if( strcmp(zMethod, "xFrames")==0 ){
    eFailing = FAILING_FRAMES;
  }else if( strcmp(zMethod, "xUndo")==0 ){
    eFailing = FAILING_UNDO;
  }else if( strcmp(zMethod, "xEnd")==0 ){
    eFailing = FAILING_END;
  }else{
    Tcl_AppendResult(interp, "unknown WAL replication method", (char*)0);
    return TCL_ERROR;
  }

  /* Error code */
  zError = Tcl_GetString(objv[2]);
  if( strcmp(zError, "NOT_LEADER")==0 ){
    rc = SQLITE_IOERR_NOT_LEADER;
  }else if( strcmp(zError, "LEADERSHIP_LOST")==0 ){
    rc = SQLITE_IOERR_LEADERSHIP_LOST;
  }else{
    Tcl_AppendResult(interp, "unknown error", (char*)0);
    return TCL_ERROR;
  }

  testWalReplicationContext.eFailing = eFailing;
  testWalReplicationContext.rc = rc;

  /* Number of failures */
  if( objc==4 ){
    if( Tcl_GetIntFromObj(interp, objv[3], &iFailures) ) return TCL_ERROR;
    testWalReplicationContext.iFailures = iFailures;
  }
  
  return TCL_OK;
}

/*
** tclcmd: sqlite3_wal_replication_frame_info N
**
** Return information about the N'th oldest frame that was handled by
** testWalReplicationFrames since the last global context reset.
**
** If N is 0, information about the most recent frame is returned.
*/
static int SQLITE_TCLAPI test_wal_replication_frame_info(
  void * clientData,
  Tcl_Interp *interp,
  int objc,
  Tcl_Obj *CONST objv[]
){
  int i;
  int n;
  testWalReplicationFrameInfo *pFrame = testWalReplicationContext.pFrameList;
  char zSzPage[32];
  char zPgno[32];
  char zPrev[32];

  if( objc!=2 ){
    Tcl_WrongNumArgs(interp, 1, objv, "N");
    return TCL_ERROR;
  }

  if( Tcl_GetIntFromObj(interp, objv[1], &n) ) return TCL_ERROR;

  for(i=0; i<n; i++){
    if( !pFrame ){
      break;
    }
    pFrame = pFrame->pNext;
  }

  if( !pFrame ){
    Tcl_AppendResult(interp, "no such frame", (char*)0);
    return TCL_ERROR;
  }

  sqlite3_snprintf(sizeof(zSzPage), zSzPage, "%d ", pFrame->szPage);
  sqlite3_snprintf(sizeof(zPgno), zPgno, "%d ", pFrame->pgno);
  sqlite3_snprintf(sizeof(zPrev), zPrev, "%d", pFrame->iPrev);

  Tcl_AppendResult(interp, zSzPage, zPgno, zPrev, (char*)0);

  return TCL_OK;
}

/*
** tclcmd: sqlite3_wal_replication_enabled HANDLE SCHEMA
**
** Return "true" if WAL replication is enabled on the given database, "false"
** otherwise.
**
** If leader replication is enabled, the name of the implementation used is also
** returned.
*/
static int SQLITE_TCLAPI test_wal_replication_enabled(
  void * clientData,
  Tcl_Interp *interp,
  int objc,
  Tcl_Obj *CONST objv[]
){
  int rc;
  sqlite3 *db;
  const char *zSchema;
  int bEnabled;
  sqlite3_wal_replication *pReplication;
  char *zEnabled;
  const char *zReplication = 0;
  char zBuf[32];

  if( objc!=3 ){
    Tcl_WrongNumArgs(interp, 1, objv, "HANDLE SCHEMA");
    return TCL_ERROR;
  }

  if( getDbPointer(interp, Tcl_GetString(objv[1]), &db) ){
    return TCL_ERROR;
  }
  zSchema = Tcl_GetString(objv[2]);

  rc = sqlite3_wal_replication_enabled(db, zSchema, &bEnabled, &pReplication);

  if( rc!=SQLITE_OK ){
    Tcl_AppendResult(interp, sqlite3ErrName(rc), (char*)0);
    return TCL_ERROR;
  }

  if( bEnabled ){
    zEnabled = "true";
    if( pReplication ){
      zReplication = pReplication->zName;
    }
  }else{
    zEnabled = "false";
  }

  if( zReplication ){
    sqlite3_snprintf(sizeof(zBuf), zBuf, " %s", zReplication);
  }else{
    zBuf[0] = 0;
  }

  Tcl_AppendResult(interp, zEnabled, zBuf, (char*)0);

  return TCL_OK;
}

/*
** tclcmd: sqlite3_wal_replication_leader HANDLE SCHEMA ?NAME?
**
** Enable leader WAL replication for the given connection/schema, using the stub
** WAL replication implementation defined in this file, or the one registered
** under NAME if given.
*/
static int SQLITE_TCLAPI test_wal_replication_leader(
  void * clientData,
  Tcl_Interp *interp,
  int objc,
  Tcl_Obj *CONST objv[]
){
  int rc;
  sqlite3 *db;
  const char *zSchema;
  const char *zReplication = "test";
  void *pArg = (void*)(&testWalReplicationContext);

  if( objc!=3 && objc!=4 ){
    Tcl_WrongNumArgs(interp, 4, objv, "HANDLE SCHEMA ?NAME?");
    return TCL_ERROR;
  }

  if( getDbPointer(interp, Tcl_GetString(objv[1]), &db) ){
    return TCL_ERROR;
  }
  zSchema = Tcl_GetString(objv[2]);

  if( objc==4 ){
    zReplication = Tcl_GetString(objv[3]);
  }

  /* Reset any previous global context state */
  testWalReplicationContextReset();

  rc = sqlite3_wal_replication_leader(db, zSchema, zReplication, pArg);

  if( rc!=SQLITE_OK ){
    Tcl_AppendResult(interp, sqlite3ErrName(rc), (char*)0);
    return TCL_ERROR;
  }

  return TCL_OK;
}

/*
** tclcmd: sqlite3_wal_replication_follower HANDLE SCHEMA
**
** Enable follower WAL replication for the given connection/schema. The global
** test replication context will be set to point to this connection/schema and
** WAL events will be replicated to it.
*/
static int SQLITE_TCLAPI test_wal_replication_follower(
  void * clientData,
  Tcl_Interp *interp,
  int objc,
  Tcl_Obj *CONST objv[]
){
  int rc;
  sqlite3 *db;
  const char *zSchema;

  if( objc!=3 ){
    Tcl_WrongNumArgs(interp, 3, objv, "HANDLE SCHEMA");
    return TCL_ERROR;
  }

  if( getDbPointer(interp, Tcl_GetString(objv[1]), &db) ){
    return TCL_ERROR;
  }
  zSchema = Tcl_GetString(objv[2]);

  rc = sqlite3_wal_replication_follower(db, zSchema);

  if( rc!=SQLITE_OK ){
    Tcl_AppendResult(interp, sqlite3ErrName(rc), (char*)0);
    return TCL_ERROR;
  }

  testWalReplicationContext.db = db;
  testWalReplicationContext.zSchema = zSchema;

  return TCL_OK;
}

/*
** tclcmd: sqlite3_wal_replication_none HANDLE SCHEMA
**
** Disable leader or follower WAL replication for the given connection/schema.
*/
static int SQLITE_TCLAPI test_wal_replication_none(
  void * clientData,
  Tcl_Interp *interp,
  int objc,
  Tcl_Obj *CONST objv[]
){
  int rc;
  sqlite3 *db;
  const char *zSchema;

  if( objc!=3 ){
    Tcl_WrongNumArgs(interp, 3, objv, "HANDLE SCHEMA");
    return TCL_ERROR;
  }

  if( getDbPointer(interp, Tcl_GetString(objv[1]), &db) ){
    return TCL_ERROR;
  }
  zSchema = Tcl_GetString(objv[2]);

  rc = sqlite3_wal_replication_none(db, zSchema);

  if( rc!=SQLITE_OK ){
    Tcl_AppendResult(interp, sqlite3ErrName(rc), (char*)0);
    return TCL_ERROR;
  }

  return TCL_OK;
}

/*
** tclcmd: sqlite3_wal_replication_checkpoint HANDLE SCHEMA
**
** Checkpoint a database in follower WAL replication mode, using the
** SQLITE_CHECKPOINT_TRUNCATE checkpoint mode.
*/
static int SQLITE_TCLAPI test_wal_replication_checkpoint(
  void * clientData,
  Tcl_Interp *interp,
  int objc,
  Tcl_Obj *CONST objv[]
){
  int rc;
  sqlite3 *db;
  const char *zSchema;
  int nLog;
  int nCkpt;

  if( objc!=3 ){
    Tcl_WrongNumArgs(interp, 1, objv,
        "HANDLE SCHEMA");
    return TCL_ERROR;
  }

  if( getDbPointer(interp, Tcl_GetString(objv[1]), &db) ){
    return TCL_ERROR;
  }
  zSchema = Tcl_GetString(objv[2]);

  rc = sqlite3_wal_replication_checkpoint(db, zSchema,
      SQLITE_CHECKPOINT_TRUNCATE, &nLog, &nCkpt);

  if( rc!=SQLITE_OK ){
    Tcl_AppendResult(interp, sqlite3ErrName(rc), (char*)0);
    return TCL_ERROR;
  }
  if( nLog!=0 ){
    Tcl_AppendResult(interp, "the WAL was not truncated", (char*)0);
    return TCL_ERROR;
  }
  if( nCkpt!=0 ){
    Tcl_AppendResult(interp, "only some frames were checkpointed", (char*)0);
    return TCL_ERROR;
  }

  return TCL_OK;
}

/*
** This routine registers the custom TCL commands defined in this
** module.  This should be the only procedure visible from outside
** of this module.
*/
int Sqlitetestwalreplication_Init(Tcl_Interp *interp){
  Tcl_CreateObjCommand(interp, "sqlite3_wal_replication_find",
          test_wal_replication_find,0,0);
  Tcl_CreateObjCommand(interp, "sqlite3_wal_replication_register",
          test_wal_replication_register,0,0);
  Tcl_CreateObjCommand(interp, "sqlite3_wal_replication_unregister",
          test_wal_replication_unregister,0,0);
  Tcl_CreateObjCommand(interp, "sqlite3_wal_replication_error",
          test_wal_replication_error,0,0);
  Tcl_CreateObjCommand(interp, "sqlite3_wal_replication_frame_info",
          test_wal_replication_frame_info,0,0);
  Tcl_CreateObjCommand(interp, "sqlite3_wal_replication_enabled",
          test_wal_replication_enabled,0,0);
  Tcl_CreateObjCommand(interp, "sqlite3_wal_replication_leader",
          test_wal_replication_leader,0,0);
  Tcl_CreateObjCommand(interp, "sqlite3_wal_replication_follower",
          test_wal_replication_follower,0,0);
  Tcl_CreateObjCommand(interp, "sqlite3_wal_replication_none",
          test_wal_replication_none,0,0);
  Tcl_CreateObjCommand(interp, "sqlite3_wal_replication_checkpoint",
          test_wal_replication_checkpoint,0,0);
  return TCL_OK;
}
#endif /* SQLITE_ENABLE_WAL_REPLICATION */
