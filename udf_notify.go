package main

/*
#cgo CFLAGS: -I/usr/include/mysql -DMYSQL_DYNAMIC_PLUGIN -DMYSQL_ABI_CHECK
#include <stdio.h>
#include <mysql.h>
#include <string.h>
#include "http_notify_plugin.h"

static int is_arg_string(UDF_ARGS *args,int arg_num) {
	if (args->arg_count > arg_num &&
		args->arg_type[arg_num] == STRING_RESULT) {
		return 1;
	}
	return 0;
}
static char* get_arg_val(UDF_ARGS *args,int arg_num) {
	if (args->arg_count > arg_num) {
		return args->args[arg_num];
	}
}


*/
import "C"

import (
	"bytes"
	"container/list"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"encoding/json"
)

type (
	worker struct {
		endoint_url      string // Endpoint url
		endoint_username string
		endoint_password string
		shutdown         chan bool
		events           *list.List // Events to Publish
		qlock            sync.Mutex // Publish lock
	}

	Notify struct {
		Method  string
		Route   string
		Payload []byte
	}

	entity struct {
		Id uint64 `json:"id"`
	}
)

const (
	eInvalidArgCount      = "Invalid number of argumets.\n"
	eInvalidRequestMethod = "Invalid request method %s. Allowed methods %s \n"
	eInvalidStringArg     = "Argument %d need to be string \n"
	defaultPoolSize       = 10
)

var w *worker

func init() {
	w = &worker{
		events:   list.New(),
		shutdown: make(chan bool),
	}
}

//export http_notify_plugin_init
func http_notify_plugin_init() C.int {

	// Setup
	url := C.GoString(C.endpoint_gvar)
	username := C.GoString(C.username_gvar)
	password := C.GoString(C.password_gvar)

	// Start Background job
	w.init(url, username, password)

	return 0
}

//export http_notify_plugin_deinit
func http_notify_plugin_deinit() C.int {
	w.shutdown <- true
	log.Printf("http_notify Unloaded !")

	return 0
}

//export http_notify_init
func http_notify_init(
	initid *C.UDF_INIT,
	args *C.UDF_ARGS,
	message *C.char,
) C.my_bool {

	// Params Method,Route,Payload
	if args.arg_count < 3 || args.arg_count > 3 {
		C.strcpy(message, C.CString(eInvalidArgCount))
		return 1
	}

	if C.is_arg_string(args, 0) == 0 {
		C.strcpy(message, C.CString(fmt.Sprintf(eInvalidStringArg, 1)))
		return 1
	}

	reqMethod := strings.ToUpper(C.GoString(C.get_arg_val(args, 0)))
	allowMehods := []string{"POST", "PUT", "DELETE"}
	for _, method := range allowMehods {
		if method == reqMethod {
			goto ValidMethod
		}
	}

	// Fail if not found
	C.strcpy(message, C.CString(fmt.Sprintf(eInvalidRequestMethod, reqMethod, strings.Join(allowMehods, "|"))))
	return 1

ValidMethod:
	if C.is_arg_string(args, 1) == 0 {
		C.strcpy(message, C.CString(fmt.Sprintf(eInvalidStringArg, 2)))
		return 1
	}

	if C.is_arg_string(args, 2) == 0 {
		C.strcpy(message, C.CString(fmt.Sprintf(eInvalidStringArg, 3)))
		return 1
	}

	return 0
}

//export http_notify
func http_notify(
	initid *C.UDF_INIT,
	args *C.UDF_ARGS,
	result *C.char,
	length *C.ulong,
	is_null *C.char,
	error *C.char,
) *C.char {

	w.qlock.Lock()
	defer w.qlock.Unlock()

	w.events.PushFront(&Notify{
		Method:  strings.ToUpper(C.GoString(C.get_arg_val(args, 0))),
		Route:   C.GoString(C.get_arg_val(args, 1)),
		Payload: []byte(C.GoString(C.get_arg_val(args, 2))),
	})

	return nil
}

func (w *worker) init(url, username, password string) {
	w.qlock.Lock()
	defer w.qlock.Unlock()

	w.endoint_url = url
	w.endoint_username = username
	w.endoint_password = password

	// TODO: Load queue from disk

	go func() {
		for {
			select {
			case <-w.shutdown:
				// TODO: Save queue to disk
				log.Printf("Receive shutdown")
				return

			case <-time.After(500 * time.Millisecond):
				w.qlock.Lock()

				events := w.events.Len()

				if events > 0 {
					log.Printf("Curent queue has %d messages !", events)
					// Limit to 10 evemnts
					if events > defaultPoolSize {
						events = defaultPoolSize
					}

					for i := 0; i < events; i++ {
						rawEvent := w.events.Back()
						w.events.Remove(rawEvent)

						event, ok := rawEvent.Value.(*Notify)
						if !ok {
							log.Printf("Invalid message type %#v", rawEvent)
							continue
						}
						// Notify Thread
						go func(e *Notify) {
							apiEndpoint := w.endoint_url + e.Route

							var reqPayload io.Reader

							switch e.Method {
							case "POST":
								reqPayload = bytes.NewBuffer(e.Payload)

							case "PUT":
								var entity *entity
								if err := json.Unmarshal(e.Payload, &entity); err != nil {
									log.Printf("Failed to unmarshal payload %s", err)
									return
								}

								reqPayload = bytes.NewBuffer(e.Payload)
								apiEndpoint = fmt.Sprintf("%s/%d", apiEndpoint, entity.Id)

							case "DELETE":
								var entity *entity
								if err := json.Unmarshal(e.Payload, &entity); err != nil {
									log.Printf("Failed to unmarshal payload %s", err)
									return
								}
								apiEndpoint = fmt.Sprintf("%s/%d", apiEndpoint, entity.Id)
							}

							request, err := http.NewRequest(e.Method, apiEndpoint, reqPayload)
							request.Header.Set("User-Agent", "MYSQL-HTTP-NOTIFY/1.0")
							request.Header.Set("Accept", "application/json")
							request.Header.Set("Content-Type", "application/json")

							// Send basic auth if creds are setup
							if w.endoint_username != "" && w.endoint_password != "" {
								request.SetBasicAuth(w.endoint_username, w.endoint_password)
							}

							retry, client := 0, &http.Client{}
						rety_request:
							response, err := client.Do(request)
							if err != nil {
								if retry < 3 {
									retry += 1
									time.Sleep(3 * time.Second)
									goto rety_request
								}

								// Requeue : Network issues
								w.qlock.Lock()
								defer w.qlock.Unlock()
								w.events.PushFront(e)

								return
							}
							defer response.Body.Close()

							resPayload, err := ioutil.ReadAll(response.Body)
							if err != nil {
								log.Printf("Fail to read response for %s %s PAYLOAD: %s\n Reason: %s", e.Method, apiEndpoint, e.Payload, err)
								return
							}

							if response.StatusCode >= 300 {
								log.Printf("%s %s (%s)\nPAYLOAD:%s\nRESPONSE: %s", e.Method, apiEndpoint, response.Status, e.Payload, resPayload)
								return
							}

						}(event)

					}
				}
				w.qlock.Unlock()
			}
		}
	}()
}

func main() {}
