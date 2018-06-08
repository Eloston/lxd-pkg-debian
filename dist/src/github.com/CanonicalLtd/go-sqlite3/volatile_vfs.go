package sqlite3

/*
#include <string.h>
#ifndef USE_LIBSQLITE3
#include <sqlite3-binding.h>
#else
#include <sqlite3.h>
#endif
#include <sys/time.h>
#include <unistd.h>
#include <stdio.h>
#include <errno.h>
#include <stdlib.h>

//
// The maximum pathname length supported by this VFS.
//
#define MAXPATHNAME 512

// SQLite VFS Go implementation.
int volatileOpen(int iVfs, char *zName, sqlite3_file *pFile, int flags, int *pOutFlags);
int volatileDelete(int iVfs, char *zName);
int volatileAccess(int iVfs, char *zName, int flags, int *pResOut);
int volatileRandomness(int nBuf, char *zBuf);
int volatileSleep(int microseconds);
int volatileGetLastError(int iVfs);

// SQLite file Go implementation.
int volatileClose(int iVfs, int iFd);
int volatileRead(int iVfs, int iFd, void *zBuf, int iAmt, sqlite_int64 iOfst);
int volatileWrite(int iVfs, int iFd, void *zBuf, int iAmt, sqlite_int64 iOfst);
int volatileTruncate(int iVfs, int iFd, sqlite_int64 size);
int volatileFileSize(int iVfs, int iFd, sqlite_int64 *pSize);
int volatileLock(int iVfs, int iFd, int eLock);
int volatileUnlock(int iVfs, int iFd, int eLock);
int volatileCheckReservedLock(int iVfs, int iFd, int *pResOut);
int volatileShmMap(int iVfs, int iFd, int iRegion, int szRegion, int bExtend, void **pp);
int volatileShmUnmap(int iVfs, int iFd, int deleteFlag);

typedef struct sqlite3VolatileFile sqlite3VolatileFile;
struct sqlite3VolatileFile {
  sqlite3_file base; // Base class. Must be first.
  int iVfs;          // Handle to a volatileVFS instance.
  int iFd;           // Handle to an open volatileFile instance.
};

static int sqlite3VolatileClose(sqlite3_file *pFile){
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;
  return volatileClose(p->iVfs, p->iFd);
}

static int sqlite3VolatileRead(
  sqlite3_file *pFile,
  void *zBuf,
  int iAmt,
  sqlite_int64 iOfst
){
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;
  return volatileRead(p->iVfs, p->iFd, zBuf, iAmt, iOfst);
}

static int sqlite3VolatileWrite(
  sqlite3_file *pFile,
  const void *zBuf,
  int iAmt,
  sqlite_int64 iOfst
){
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;
  return volatileWrite(p->iVfs, p->iFd, (void*)zBuf, iAmt, iOfst);
}

static int sqlite3VolatileTruncate(sqlite3_file *pFile, sqlite_int64 size){
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;
  return volatileTruncate(p->iVfs, p->iFd, size);
}

static int sqlite3VolatileSync(sqlite3_file *pFile, int flags){
  return SQLITE_OK;
}

static int sqlite3VolatileFileSize(sqlite3_file *pFile, sqlite_int64 *pSize){
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;
  return volatileFileSize(p->iVfs, p->iFd, pSize);
}

static int sqlite3VolatileLock(sqlite3_file *pFile, int eLock){
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;
  return volatileLock(p->iVfs, p->iFd, eLock);
}

static int sqlite3VolatileUnlock(sqlite3_file *pFile, int eLock){
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;
  return volatileUnlock(p->iVfs, p->iFd, eLock);
}

static int sqlite3VolatileCheckReservedLock(sqlite3_file *pFile, int *pResOut){
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;
  return volatileCheckReservedLock(p->iVfs, p->iFd, pResOut);
}

static int sqlite3VolatileFileControl(sqlite3_file *pFile, int op, void *pArg){
  if( op==SQLITE_FCNTL_PRAGMA ){
    // This is needed in order for pragmas to work. See the xFileControl docstring
    // in sqlite.h.in.
    //
    // TODO: there are other op codes that should be handled. Also, xFileControl
    //       should return SQLITE_OK if the pragma is already applied.
    return SQLITE_NOTFOUND;
  }
  return SQLITE_OK;
}

static int sqlite3VolatileSectorSize(sqlite3_file *pFile){
  return 0;
}

static int sqlite3VolatileDeviceCharacteristics(sqlite3_file *pFile){
  return 0;
}

static int sqlite3VolatileShmMap(
  sqlite3_file *pFile,            // Handle open on database file
  int iRegion,                    // Region to retrieve
  int szRegion,                   // Size of regions
  int bExtend,                    // True to extend file if necessary
  void volatile **pp              // OUT: Mapped memory
){
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;
  return volatileShmMap(p->iVfs, p->iFd, iRegion, szRegion, bExtend, (void**)pp);
}

static int sqlite3VolatileShmLock(sqlite3_file *pFile, int ofst, int n, int flags){
  // This is a no-op since shared-memory locking is relevant only for
  // inter-process concurrency. See also the unix-excl branch from upstream
  // (git commit cda6b3249167a54a0cf892f949d52760ee557129).
  return SQLITE_OK;
}

static void sqlite3VolatileShmBarrier(sqlite3_file *pFile){
  // This is a no-op since we expect SQLite to be compiled with mutex
  // support (i.e. SQLITE_MUTEX_OMIT or SQLITE_MUTEX_NOOP are *not*
  // defined, see sqliteInt.h).
}

static int sqlite3VolatileShmUnmap(sqlite3_file *pFile, int deleteFlag){
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;
  return volatileShmUnmap(p->iVfs, p->iFd, deleteFlag);
}

static int sqlite3VolatileOpen(
  sqlite3_vfs *pVfs,              // VFS
  const char *zName,              // File to open, or 0 for a temp file
  sqlite3_file *pFile,            // Pointer to DemoFile struct to populate
  int flags,                      // Input SQLITE_OPEN_XXX flags
  int *pOutFlags                  // Output SQLITE_OPEN_XXX flags (or NULL)
){
  int rc = SQLITE_OK;
  int vfs = *(int*)(pVfs->pAppData);
  sqlite3VolatileFile *p = (sqlite3VolatileFile*)pFile;

  rc = volatileOpen(vfs, (char*)zName, pFile, flags, pOutFlags);
  if( rc!= SQLITE_OK ){
    p->base.pMethods = 0; // This signal SQLite to not call Close().
    return rc;
  }

  static const sqlite3_io_methods io = {
    2,                                       // iVersion
    sqlite3VolatileClose,                    // xClose
    sqlite3VolatileRead,                     // xRead
    sqlite3VolatileWrite,                    // xWrite
    sqlite3VolatileTruncate,                 // xTruncate
    sqlite3VolatileSync,                     // xSync
    sqlite3VolatileFileSize,                 // xFileSize
    sqlite3VolatileLock,                     // xLock
    sqlite3VolatileUnlock,                   // xUnlock
    sqlite3VolatileCheckReservedLock,        // xCheckReservedLock
    sqlite3VolatileFileControl,              // xFileControl
    sqlite3VolatileSectorSize,               // xSectorSize
    sqlite3VolatileDeviceCharacteristics,    // xDeviceCharacteristics
    sqlite3VolatileShmMap,                   // xShmMap
    sqlite3VolatileShmLock,                  // xShmLock
    sqlite3VolatileShmBarrier,               // xShmBarrier
    sqlite3VolatileShmUnmap                  // xShmUnmap
  };

  p->base.pMethods = &io;

  return SQLITE_OK;
}

static int sqlite3VolatileDelete(sqlite3_vfs *pVfs, const char *zPath, int dirSync){
  return volatileDelete(*(int*)(pVfs->pAppData), (char*)zPath);
}

static int sqlite3VolatileAccess(
  sqlite3_vfs *pVfs,
  const char *zPath,
  int flags,
  int *pResOut
){
  return volatileAccess(*(int*)(pVfs->pAppData), (char*)zPath, flags, pResOut);
}

static int sqlite3VolatileFullPathname(
  sqlite3_vfs *pVfs,              // VFS
  const char *zPath,              // Input path (possibly a relative path)
  int nPathOut,                   // Size of output buffer in bytes
  char *zPathOut                  // Pointer to output buffer
){
  // Just return the path unchanged.
  sqlite3_snprintf(nPathOut, zPathOut, "%s", zPath);
  return SQLITE_OK;
}

static void* sqlite3VolatileDlOpen(sqlite3_vfs *pVfs, const char *zPath){
  return 0;
}

static void sqlite3VolatileDlError(sqlite3_vfs *pVfs, int nByte, char *zErrMsg){
  sqlite3_snprintf(nByte, zErrMsg, "Loadable extensions are not supported");
  zErrMsg[nByte-1] = '\0';
}

static void (*sqlite3VolatileDlSym(sqlite3_vfs *pVfs, void *pH, const char *z))(void){
  return 0;
}

static void sqlite3VolatileDlClose(sqlite3_vfs *pVfs, void *pHandle){
  return;
}

static int sqlite3VolatileRandomness(sqlite3_vfs *pVfs, int nByte, char *zByte){
  return volatileRandomness(nByte, zByte);
}

static int sqlite3VolatileSleep(sqlite3_vfs *NotUsed, int microseconds){
  // Sleep in Go, to avoid the scheduler unconditionally preempting the
  // SQLite API call being invoked.
  return volatileSleep(microseconds);
}

static int sqlite3VolatileCurrentTimeInt64(sqlite3_vfs *pVfs, sqlite3_int64 *piNow){
  static const sqlite3_int64 unixEpoch = 24405875*(sqlite3_int64)8640000;
  struct timeval sNow;
  (void)gettimeofday(&sNow, 0);
  *piNow = unixEpoch + 1000*(sqlite3_int64)sNow.tv_sec + sNow.tv_usec/1000;
  return SQLITE_OK;
}

static int sqlite3VolatileCurrentTime(sqlite3_vfs *pVfs, double *piNow){
  // TODO: check if it's always safe to cast a double* to a sqlite3_int64*.
  return sqlite3VolatileCurrentTimeInt64(pVfs, (sqlite3_int64*)piNow);
}

static int sqlite3VolatileGetLastError(sqlite3_vfs *pVfs, int NotUsed2, char *NotUsed3){
  return volatileGetLastError(*(int*)(pVfs->pAppData));
}

static int sqlite3VolatileRegister(char *zName, int iVfs, sqlite3_vfs **ppVfs) {
  sqlite3_vfs* pRet;
  void *pAppData;

  pRet = (sqlite3_vfs*)sqlite3_malloc(sizeof(sqlite3_vfs));
  if( !pRet ){
    return SQLITE_NOMEM;
  }
  pAppData = (void*)sqlite3_malloc(sizeof(int));
  if( !pAppData ){
    return SQLITE_NOMEM;
  }
  *(int*)(pAppData) = iVfs;

  pRet->iVersion =          2;
  pRet->szOsFile =          sizeof(sqlite3VolatileFile);
  pRet->mxPathname =        MAXPATHNAME;
  pRet->pNext =             0;
  pRet->zName =             (const char*)zName;
  pRet->pAppData =          pAppData;
  pRet->xOpen =             sqlite3VolatileOpen;
  pRet->xDelete =           sqlite3VolatileDelete;
  pRet->xAccess =           sqlite3VolatileAccess;
  pRet->xFullPathname =     sqlite3VolatileFullPathname;
  pRet->xDlOpen =           sqlite3VolatileDlOpen;
  pRet->xDlError =          sqlite3VolatileDlError;
  pRet->xDlSym =            sqlite3VolatileDlSym;
  pRet->xDlClose =          sqlite3VolatileDlClose;
  pRet->xRandomness =       sqlite3VolatileRandomness;
  pRet->xSleep =            sqlite3VolatileSleep;
  pRet->xCurrentTime =      sqlite3VolatileCurrentTime;
  pRet->xGetLastError =     sqlite3VolatileGetLastError;
  pRet->xCurrentTimeInt64 = sqlite3VolatileCurrentTimeInt64;

  sqlite3_vfs_register(pRet, 0);

  *ppVfs = pRet;

  return SQLITE_OK;
}

static void sqlite3VolatileUnregister(sqlite3_vfs* pVfs) {
  sqlite3_vfs_unregister(pVfs);
  sqlite3_free(pVfs->pAppData);
  sqlite3_free(pVfs);
}

*/
import "C"
import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/pkg/errors"
)

// RegisterVolatileFileSystem registers a new volatile VFS under the given
// name.
func RegisterVolatileFileSystem(name string) *VolatileFileSystem {
	volatileVFSLock.Lock()
	defer volatileVFSLock.Unlock()

	iFs := volatileVFSHandles
	volatileVFSHandles++

	vfs := newVolatileVFS()
	volatileVFSs[iFs] = vfs

	zName := C.CString(name)
	rc := C.sqlite3VolatileRegister(zName, iFs, &vfs.pVfs)
	if rc != C.SQLITE_OK {
		panic("out of memory")
	}

	return &VolatileFileSystem{
		zName: zName,
		vfs:   vfs,
	}
}

// UnregisterVolatileFileSystem unregisters the given volatile VFS.
func UnregisterVolatileFileSystem(fs *VolatileFileSystem) {
	volatileVFSLock.Lock()
	defer volatileVFSLock.Unlock()

	for iFs := range volatileVFSs {
		if volatileVFSs[iFs] == fs.vfs {
			C.sqlite3VolatileUnregister(fs.vfs.pVfs)
			C.free(unsafe.Pointer(fs.zName))
			delete(volatileVFSs, iFs)
			return
		}
	}

	panic("unknown volatile file system")
}

// Global registry of volatileFileSystem instances.
var volatileVFSLock sync.RWMutex
var volatileVFSs = make(map[C.int]*volatileVFS)
var volatileVFSHandles C.int

// VolatileFileSystem exports APIs to inspect the internal VFS implementation.
type VolatileFileSystem struct {
	zName *C.char      // C string used for registration.
	vfs   *volatileVFS // VFS implementation.
}

// Name returns the VFS name this volatile file system was registered with.
func (fs *VolatileFileSystem) Name() string {
	return C.GoString(fs.zName)
}

// ReadFile returns a copy of the content of the volatile file with the given
// name.
//
// If the file does not exists, an error is returned.
func (fs *VolatileFileSystem) ReadFile(name string) ([]byte, error) {
	file, rc := fs.vfs.FileByName(name)
	if rc != C.SQLITE_OK {
		return nil, Error{
			Code:         ErrIoErr,
			ExtendedCode: ErrIoErrRead,
		}
	}

	return file.data[:], nil
}

// CreateFile adds a new volatile file with the given name and content.
//
// If the file already exists, an error is returned.
func (fs *VolatileFileSystem) CreateFile(name string, data []byte) error {
	var flags C.int = C.SQLITE_OPEN_EXCLUSIVE | C.SQLITE_OPEN_CREATE

	iFd, rc := fs.vfs.Open(name, flags)
	if rc != C.SQLITE_OK {
		return Error{
			Code:         ErrIoErr,
			ExtendedCode: ErrNoExtended(rc),
		}
	}

	file, _ := fs.vfs.FileByFD(iFd)
	file.mu.Lock()
	file.data = data
	file.mu.Unlock()

	return nil
}

// FileSize returns the size of the file with the given name.
func (fs *VolatileFileSystem) FileSize(name string) (int, error) {
	file, rc := fs.vfs.FileByName(name)
	if rc != C.SQLITE_OK {
		return -1, Error{
			Code:         ErrIoErr,
			ExtendedCode: ErrIoErrRead,
		}
	}

	return file.Size(), nil
}

// Remove the volatile file with the given name.
func (fs *VolatileFileSystem) Remove(name string) error {
	rc := fs.vfs.Delete(name)
	if rc != C.SQLITE_OK {
		return Error{
			Code:         ErrIoErr,
			ExtendedCode: ErrNoExtended(rc),
		}
	}
	return nil
}

// Dump the content of all volatile files to the given directory.
func (fs *VolatileFileSystem) Dump(dir string) error {
	fs.vfs.mu.Lock()
	defer fs.vfs.mu.Unlock()

	for name, file := range fs.vfs.files {
		if err := volatileDumpFile(file.data, dir, name); err != nil {
			return errors.Wrapf(err, "failed to dump file %s", name)
		}
	}

	return nil
}

// Implements the SQLite VFS API storing files in-memory.
type volatileVFS struct {
	mu     sync.RWMutex
	pVfs   *C.sqlite3_vfs
	files  map[string]*volatileFile // Map file names to file objects.
	fds    map[C.int]*volatileFile  // Map C-land open file numbers to files objects.
	serial C.int                    // Serial number for file numbers, increasing monotonically.
	errno  C.int                    // Last error.
}

func newVolatileVFS() *volatileVFS {
	return &volatileVFS{
		files: make(map[string]*volatileFile),
		fds:   make(map[C.int]*volatileFile),
	}
}

// Open a new volatile file.
func (vfs *volatileVFS) Open(name string, flags C.int) (C.int, C.int) {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	file, ok := vfs.files[name]

	// If file exists, and the exclusive flag is on, the return an error.
	//
	// From sqlite3.h.in:
	//
	//   The SQLITE_OPEN_EXCLUSIVE flag is always used in conjunction with
	//   the SQLITE_OPEN_CREATE flag, which are both directly analogous to
	//   the O_EXCL and O_CREAT flags of the POSIX open() API.  The
	//   SQLITE_OPEN_EXCLUSIVE flag, when paired with the
	//   SQLITE_OPEN_CREATE, is used to indicate that file should always be
	//   created, and that it is an error if it already exists.  It is not
	//   used to indicate the file should be opened for exclusive access.
	//
	if ok && (flags&C.SQLITE_OPEN_EXCLUSIVE) != 0 {
		vfs.errno = C.EEXIST
		return -1, C.SQLITE_CANTOPEN
	}

	if !ok {
		// Check the create flag.
		if (flags & C.SQLITE_OPEN_CREATE) == 0 {
			vfs.errno = C.ENOENT
			return -1, C.SQLITE_CANTOPEN
		}
		// This is a new file.
		file = newVolatileFile(name)
		vfs.files[name] = file

	}

	// Create a new file handle.
	iFd := vfs.serial
	vfs.fds[iFd] = file
	vfs.serial++

	return iFd, C.SQLITE_OK
}

// Close new volatile file.
func (vfs *volatileVFS) Close(iFd C.int) C.int {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	if _, ok := vfs.fds[iFd]; !ok {
		vfs.errno = C.EBADF
		return C.SQLITE_IOERR_CLOSE
	}

	delete(vfs.fds, iFd)

	return C.SQLITE_OK
}

// Delete new volatile file.
func (vfs *volatileVFS) Delete(name string) C.int {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	file, ok := vfs.files[name]
	if !ok {
		vfs.errno = C.ENOENT
		return C.SQLITE_IOERR_DELETE_NOENT
	}

	// Check that there are no consumers of this file.
	for iFd := range vfs.fds {
		if vfs.fds[iFd] == file {
			vfs.errno = C.EBUSY
			return C.SQLITE_IOERR_DELETE
		}
	}

	delete(vfs.files, name)

	return C.SQLITE_OK
}

// Access returns true if the file exists.
func (vfs *volatileVFS) Access(name string, flags C.int) bool {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	_, ok := vfs.files[name]
	if !ok {
		vfs.errno = C.ENOENT
		return false
	}

	return true
}

// GetLastError returns the last error happened.
func (vfs *volatileVFS) GetLastError() C.int {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	return vfs.errno
}

// FileByFD returns the open volatile file with the given fd number.
func (vfs *volatileVFS) FileByFD(iFd C.int) (*volatileFile, C.int) {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	file, ok := vfs.fds[iFd]
	if !ok {
		vfs.errno = C.EBADF
		return nil, C.SQLITE_IOERR
	}

	return file, C.SQLITE_OK
}

// FileByName returns the volatile file with the given name.
func (vfs *volatileVFS) FileByName(name string) (*volatileFile, C.int) {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	file, ok := vfs.files[name]
	if !ok {
		vfs.errno = C.EBADF
		return nil, C.SQLITE_IOERR
	}

	return file, C.SQLITE_OK
}

// Hold the content of a volatile in-memory file.
type volatileFile struct {
	mu          sync.RWMutex     // Serialize access to the fields below.
	data        []byte           // Content of the file.
	shm         []unsafe.Pointer // Regions of C-allocated memory
	shmRefCount int              // Number of opened files referencing the shared memory

	// Lock counters.
	none      int
	shared    int
	reserved  int
	pending   int
	exclusive int
}

func newVolatileFile(name string) *volatileFile {
	return &volatileFile{
		data: make([]byte, 0),
		shm:  make([]unsafe.Pointer, 0),
	}
}

// Read data from the file.
func (f *volatileFile) Read(buf unsafe.Pointer, n int, offset int) C.int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var rc C.int
	rc = C.SQLITE_OK
	size := unsafe.Sizeof(byte(0))

	// From SQLite docs:
	//
	//   If xRead() returns SQLITE_IOERR_SHORT_READ it must also fill
	//   in the unread portions of the buffer with zeros.  A VFS that
	//   fails to zero-fill short reads might seem to work.  However,
	//   failure to zero-fill short reads will eventually lead to
	//   database corruption.
	//
	// So we loop through the full range.
	j := offset
	for i := 0; i < n; i++ {
		value := byte(0)
		if j < len(f.data) {
			value = f.data[j]
		} else {
			rc = C.SQLITE_IOERR_SHORT_READ
		}
		pByte := (*byte)(unsafe.Pointer(uintptr(buf) + size*uintptr(i)))
		*pByte = value
		j++
	}

	return rc
}

// Write data to the file.
func (f *volatileFile) Write(buf unsafe.Pointer, n int, offset int) C.int {
	f.mu.Lock()
	defer f.mu.Unlock()

	if offset+n >= len(f.data) {
		f.data = append(f.data, make([]byte, offset+n-len(f.data))...)
	}

	size := unsafe.Sizeof(byte(0))

	for i := 0; i < n; i++ {
		j := i + offset
		f.data[j] = *(*byte)(unsafe.Pointer(uintptr(buf) + size*uintptr(i)))
	}

	return C.SQLITE_OK
}

// Truncate the file.
func (f *volatileFile) Truncate(size int) C.int {
	f.mu.Lock()
	defer f.mu.Unlock()

	if size >= len(f.data) {
		f.data = append(f.data, make([]byte, size-len(f.data))...)
	} else {
		f.data = f.data[:size]
	}

	return C.SQLITE_OK
}

// Size returns the size of the file.
func (f *volatileFile) Size() int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return len(f.data)
}

// Lock increases the count of the given lock type.
func (f *volatileFile) Lock(lock C.int) C.int {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch lock {
	case C.SQLITE_LOCK_NONE:
		f.none++
	case C.SQLITE_LOCK_SHARED:
		f.shared++
	case C.SQLITE_LOCK_RESERVED:
		f.reserved++
	case C.SQLITE_LOCK_PENDING:
		f.pending++
	case C.SQLITE_LOCK_EXCLUSIVE:
		f.exclusive++
	default:
		return C.SQLITE_ERROR
	}

	return C.SQLITE_OK
}

// Unlock decreases the count of the given lock type.
func (f *volatileFile) Unlock(lock C.int) C.int {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch lock {
	case C.SQLITE_LOCK_NONE:
		f.none--
	case C.SQLITE_LOCK_SHARED:
		f.shared--
	case C.SQLITE_LOCK_RESERVED:
		f.reserved--
	case C.SQLITE_LOCK_PENDING:
		f.pending--
	case C.SQLITE_LOCK_EXCLUSIVE:
		f.exclusive--
	default:
		return C.SQLITE_ERROR
	}

	return C.SQLITE_OK
}

// CheckReservedLock returns true if a write lock is hold.
func (f *volatileFile) CheckReservedLock() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.reserved > 0 || f.pending > 0 || f.exclusive > 0
}

// ShmMap simulates shared memory by allocating on the C heap.
func (f *volatileFile) ShmMap(region int, size int, extend int) (unsafe.Pointer, C.int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if region < len(f.shm) {
		// The region was already allocated.
		f.shmRefCount++
		return f.shm[region], C.SQLITE_OK
	}
	if extend == 0 {
		return nil, C.SQLITE_OK
	}

	data := C.sqlite3_malloc(C.int(size))
	if data == nil {
		return nil, C.SQLITE_NOMEM
	}
	C.memset(data, C.int(0), C.size_t(size))

	f.shm = append(f.shm, data)
	f.shmRefCount++

	return data, C.SQLITE_OK
}

// ShmUnmap frees heap memory allocated by ShmMap.
func (f *volatileFile) ShmUnmap(deleteFlag int) C.int {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.shmRefCount--
	if f.shmRefCount == 0 {
		for _, data := range f.shm {
			C.sqlite3_free(data)
		}
		f.shm = f.shm[0:0]
	}

	return C.SQLITE_OK
}

//export volatileOpen
func volatileOpen(iVfs C.int, zName *C.char, pFile *C.sqlite3_file, flags C.int, pOutFlags *C.int) C.int {
	volatileVFSLock.RLock()
	defer volatileVFSLock.RUnlock()

	vfs, ok := volatileVFSs[iVfs]
	if !ok {
		return C.SQLITE_CANTOPEN
	}
	iFd, rc := vfs.Open(C.GoString(zName), flags)
	if rc != C.SQLITE_OK {
		return rc
	}

	file := (*C.sqlite3VolatileFile)(unsafe.Pointer(pFile))
	file.iFd = iFd
	file.iVfs = iVfs

	return C.SQLITE_OK
}

//export volatileDelete
func volatileDelete(iVfs C.int, zName *C.char) C.int {
	volatileVFSLock.RLock()
	defer volatileVFSLock.RUnlock()

	vfs, ok := volatileVFSs[iVfs]
	if !ok {
		return C.SQLITE_IOERR_DELETE_NOENT
	}
	return vfs.Delete(C.GoString(zName))
}

//export volatileAccess
func volatileAccess(iVfs C.int, zName *C.char, flags C.int, pResOut *C.int) C.int {
	volatileVFSLock.RLock()
	defer volatileVFSLock.RUnlock()

	vfs, ok := volatileVFSs[iVfs]
	if !ok {
		return C.SQLITE_IOERR_FSTAT
	}

	access := vfs.Access(C.GoString(zName), flags)
	if access {
		*pResOut = 1
	} else {
		*pResOut = 0
	}

	return C.SQLITE_OK
}

//export volatileRandomness
func volatileRandomness(nBuf C.int, zBuf *C.char) C.int {
	buf := make([]byte, nBuf)
	rand.Read(buf) // According to the documentation this never fails.

	start := unsafe.Pointer(zBuf)
	size := unsafe.Sizeof(*zBuf)
	for i := 0; i < int(nBuf); i++ {
		pChar := (*C.char)(unsafe.Pointer(uintptr(start) + size*uintptr(i)))
		*pChar = C.char(buf[i])
	}

	return C.SQLITE_OK
}

//export volatileSleep
func volatileSleep(microseconds C.int) C.int {
	time.Sleep(time.Duration(microseconds) * time.Microsecond)
	return microseconds
}

//export volatileGetLastError
func volatileGetLastError(iVfs C.int) C.int {
	volatileVFSLock.RLock()
	defer volatileVFSLock.RUnlock()

	vfs, ok := volatileVFSs[iVfs]
	if !ok {
		return C.SQLITE_IOERR
	}

	return vfs.GetLastError()
}

func volatileFindFile(iVfs C.int, iFd C.int) (*volatileFile, C.int) {
	volatileVFSLock.RLock()
	defer volatileVFSLock.RUnlock()

	vfs, ok := volatileVFSs[iVfs]
	if !ok {
		return nil, C.SQLITE_IOERR
	}

	return vfs.FileByFD(iFd)
}

//export volatileClose
func volatileClose(iVfs C.int, iFd C.int) C.int {
	volatileVFSLock.RLock()
	defer volatileVFSLock.RUnlock()

	vfs, ok := volatileVFSs[iVfs]
	if !ok {
		return C.SQLITE_IOERR_CLOSE
	}

	return vfs.Close(iFd)
}

//export volatileRead
func volatileRead(iVfs C.int, iFd C.int, zBuf unsafe.Pointer, iAmt C.int, iOfst C.sqlite_int64) C.int {
	file, rc := volatileFindFile(iVfs, iFd)
	if rc != C.SQLITE_OK {
		return rc
	}

	// Here we convenrt iOfst which is int64 to int, which is not
	// guarenteed to be 64-bit. This means that on 32-bit architectures
	// maximum file size will be limited to ~2G.
	rc = file.Read(zBuf, int(iAmt), int(iOfst))
	return rc
}

//export volatileWrite
func volatileWrite(iVfs C.int, iFd C.int, zBuf unsafe.Pointer, iAmt C.int, iOfst C.sqlite_int64) C.int {
	file, rc := volatileFindFile(iVfs, iFd)
	if rc != C.SQLITE_OK {
		return rc
	}

	// Here we convenrt iOfst which is int64 to int, which is not
	// guarenteed to be 64-bit. This means that on 32-bit architectures
	// maximum file size will be limited to ~2G.
	return file.Write(zBuf, int(iAmt), int(iOfst))
}

//export volatileTruncate
func volatileTruncate(iVfs C.int, iFd C.int, size C.sqlite_int64) C.int {
	file, rc := volatileFindFile(iVfs, iFd)
	if rc != C.SQLITE_OK {
		return rc
	}

	// Here we convenrt size which is int64 to int, which is not
	// guarenteed to be 64-bit. This means that on 32-bit architectures
	// maximum file size will be limited to ~2G.
	return file.Truncate(int(size))
}

//export volatileFileSize
func volatileFileSize(iVfs C.int, iFd C.int, pSize *C.sqlite3_int64) C.int {
	file, rc := volatileFindFile(iVfs, iFd)
	if rc != C.SQLITE_OK {
		return rc
	}

	// Here we convenrt size which is int64 to int, which is not
	// guarenteed to be 64-bit. This means that on 32-bit architectures
	// maximum file size will be limited to ~2G.
	*pSize = C.sqlite3_int64(file.Size())

	return C.SQLITE_OK
}

//export volatileLock
func volatileLock(iVfs C.int, iFd C.int, eLock C.int) C.int {
	file, rc := volatileFindFile(iVfs, iFd)
	if rc != C.SQLITE_OK {
		return rc
	}

	file.Lock(eLock)

	return C.SQLITE_OK
}

//export volatileUnlock
func volatileUnlock(iVfs C.int, iFd C.int, eUnlock C.int) C.int {
	file, rc := volatileFindFile(iVfs, iFd)
	if rc != C.SQLITE_OK {
		return rc
	}

	file.Unlock(eUnlock)

	return C.SQLITE_OK
}

//export volatileCheckReservedLock
func volatileCheckReservedLock(iVfs C.int, iFd C.int, pResOut *C.int) C.int {
	file, rc := volatileFindFile(iVfs, iFd)
	if rc != C.SQLITE_OK {
		return rc
	}

	if file.CheckReservedLock() {
		*pResOut = 1
	} else {
		*pResOut = 0
	}

	return C.SQLITE_OK
}

//export volatileShmMap
func volatileShmMap(iVfs C.int, iFd C.int, iRegion C.int, szRegion C.int, bExtend C.int, pp *unsafe.Pointer) C.int {
	file, rc := volatileFindFile(iVfs, iFd)
	if rc != C.SQLITE_OK {
		return rc
	}

	p, rc := file.ShmMap(int(iRegion), int(szRegion), int(bExtend))
	if rc != C.SQLITE_OK {
		return rc
	}

	*pp = p

	return C.SQLITE_OK
}

//export volatileShmUnmap
func volatileShmUnmap(iVfs C.int, iFd C.int, deleteFlag C.int) C.int {
	file, rc := volatileFindFile(iVfs, iFd)
	if rc != C.SQLITE_OK {
		return rc
	}

	return file.ShmUnmap(int(deleteFlag))
}

// Dump the content of a volatile file to the actual file system.
func volatileDumpFile(data []byte, dir string, name string) error {
	if strings.HasPrefix(name, "/") {
		return fmt.Errorf("can't dump absolute file path %s", name)
	}

	path := filepath.Join(dir, name)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errors.Wrap(err, "failed to create parent directory")
	}
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	return nil
}
