package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/lxc/lxd/shared"
)

func containersGet(d *Daemon, r *http.Request) Response {
	for {
		result, err := doContainersGet(d, d.isRecursionRequest(r))
		if err == nil {
			return SyncResponse(true, result)
		}
		if !isDbLockedError(err) {
			shared.Debugf("DBERR: containersGet: error %q\n", err)
			return InternalError(err)
		}
		// 1 s may seem drastic, but we really don't want to thrash
		// perhaps we should use a random amount
		shared.Debugf("DBERR: containersGet, db is locked\n")
		shared.PrintStack()
		time.Sleep(1 * time.Second)
	}
}

func doContainersGet(d *Daemon, recursion bool) (interface{}, error) {
	result, err := dbListContainers(d)
	if err != nil {
		return nil, err
	}

	result_string := make([]string, 0)
	result_map := make([]shared.ContainerInfo, 0)
	if err != nil {
		return []string{}, err
	}
	for _, container := range result {
		if !recursion {
			url := fmt.Sprintf("/%s/containers/%s", shared.APIVersion, container)
			result_string = append(result_string, url)
		} else {
			container, response := doContainerGet(d, container)
			if response != nil {
				continue
			}
			result_map = append(result_map, container)
		}
	}

	if !recursion {
		return result_string, nil
	} else {
		return result_map, nil
	}
}

func doContainerGet(d *Daemon, cname string) (shared.ContainerInfo, Response) {
	_, err := dbGetContainerId(d.db, cname)
	if err != nil {
		return shared.ContainerInfo{}, SmartError(err)
	}

	c, err := newLxdContainer(cname, d)
	if err != nil {
		return shared.ContainerInfo{}, SmartError(err)
	}

	var name string
	regexp := fmt.Sprintf("%s/", cname)
	length := len(regexp)
	q := "SELECT name FROM containers WHERE type=? AND SUBSTR(name,1,?)=?"
	inargs := []interface{}{cTypeSnapshot, length, regexp}
	outfmt := []interface{}{name}
	results, err := dbQueryScan(d.db, q, inargs, outfmt)
	if err != nil {
		return shared.ContainerInfo{}, SmartError(err)
	}

	var body []string

	for _, r := range results {
		name = r[0].(string)

		url := fmt.Sprintf("/%s/containers/%s/snapshots/%s", shared.APIVersion, cname, name)
		body = append(body, url)
	}

	cts, err := c.RenderState()
	if err != nil {
		return shared.ContainerInfo{}, SmartError(err)
	}

	containerinfo := shared.ContainerInfo{State: *cts,
		Snaps: body}

	return containerinfo, nil
}
