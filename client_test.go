// This package provides simple tests to ensure correctness of the tlvclient
// package.  A stub server and handlers are implemented in order to test the
// client within this package.  The following environment variables should be
// set in order to run this test.
//
// TLVCLIENT_SERVER_ADDR
//
// This environment variable should take the form "host:port"
package tlvclient_test

import (
	"bytes"
	"github.com/pdmorrow/tlvclient"
	"github.com/pdmorrow/tlvserv"
	"log"
	"net"
	"os"
	"testing"
	"time"
)

const (
	mtypeHello tlvserv.MTypeID = tlvserv.MTypeIDUserStart
	mtypeEcho  tlvserv.MTypeID = iota
)

type handlerStatus struct {
	mtype tlvserv.MTypeID
	err   error
}

var msgServer *tlvserv.TLVServ
var msgClient *tlvclient.TLVClient
var statusChan chan handlerStatus
var echoDelay time.Duration

// handleHello handles a message type of mtypeHello.  It unblocks the status
// channel when the routine completes.  This routine is provided to allow a
// test server to be instantiated.
func handleHello(conn net.Conn, data []byte) {
	var hs handlerStatus

	// Signal back via the test status channel
	hs.mtype = mtypeHello
	hs.err = nil
	statusChan <- hs
}

// handleEcho handles a message type of mtypeEcho.  It echos back the data
// received in the data parameter then  unblocks the status channel when the
// routine completes.  This routine is provided to allow a test sever to be
// instanciated.
func handleEcho(conn net.Conn, data []byte) {
	var hs handlerStatus

	// Maybe delay
	time.Sleep(echoDelay)
	// Echo back the data
	conn.Write(data)
	// Signal back via the test status channel
	hs.mtype = mtypeEcho
	hs.err = nil
	statusChan <- hs
}

// ExampleClient connects to a server listening on the address described by
// the TLVCLIENT_SERVER_ADDR environment variable.  The format of this
// environment variable is "address:port".
func ExampleClient() *tlvclient.TLVClient {
	hostAndPort, ok := os.LookupEnv("TLVCLIENT_SERVER_ADDR")
	if ok == true {
		client, err := tlvclient.Client(
			// network
			"tcp",
			// address:port
			hostAndPort)
		if err != nil {
			log.Fatal(err)
		}

		return client
	} else {
		log.Println("environment variable TLVCLIENT_SERVER_ADDR is not set")
	}

	return nil
}

// testServer() starts a test server on localhost port TLVCLIENT_SERVER_ADDR.
func testServer() *tlvserv.TLVServ {

	hostAndPort, ok := os.LookupEnv("TLVCLIENT_SERVER_ADDR")
	if ok == true {
		serv, err := tlvserv.Server(
			// "network"
			"tcp",
			// address:port
			hostAndPort,
			// Read timeout for reading payloads.
			time.Duration(5*time.Second),
			// Map of types to message handler.
			map[tlvserv.MTypeID]func(net.Conn, []byte){
				mtypeHello: handleHello,
				mtypeEcho:  handleEcho,
			})

		if err != nil {
			log.Fatal(err)
		}

		return serv
	} else {
		log.Println("environment variable TLVCLIENT_SERVER_ADDR is not set")
	}

	return nil
}

// TestMain is called in the main thread prior to any
func TestMain(m *testing.M) {
	msgServer = testServer()
	if msgServer != nil {
		// Create a channel for signalling test success or failure from
		// message handler routines.
		statusChan = make(chan handlerStatus)
		// Create the client object.
		msgClient = testClient()
		if msgClient != nil {
			// Run all the tests.
			code := m.Run()
			// Exit.
			os.Exit(code)
		} else {
			os.Exit(-1)
		}
	} else {
		os.Exit(-1)
	}
}

// TestHelloMsg connects to the server and send a hello message, the test
// passes if the handleHello routine is called by the server.
func TestHelloMsg(t *testing.T) {
	err := msgClient.WriteTLV(mtypeHello, 0, []byte{})
	if err != nil {
		t.Errorf("failed to send mtypeHello msg: %v\n", err)
	} else {
		select {
		case sm := <-statusChan:
			if sm.err != nil {
				t.Error(sm.err)
			} else {
				// Handler has been called, we should be able to read the
				// response now.
			}

		case <-time.After(10 * time.Millisecond):
			t.Errorf("timeout")
		}
	}
}

// TestEchoMsg connects to the server and send an echo message, the test
// passes if the server echos back the same data we sent it.
func TestEchoMsg(t *testing.T) {
	// Send the "echo" command.
	echoDelay = time.Duration(0)
	data := []byte("echodata")
	datalen := uint16(len(data))
	err := msgClient.WriteTLV(mtypeEcho, datalen, data)

	// Wait 10 milliseconds before declaring an error, within 10ms
	// the handler for mtypeEcho should have been called.
	if err == nil {
		select {
		case sm := <-statusChan:
			if sm.err != nil {
				t.Error(sm.err)
			} else {
				// Handler has been called, we should be able to read the
				// response now.
				resp, err := msgClient.ReadData(datalen,
					time.Duration(time.Second))
				if err != nil {
					t.Error(err)
				} else {
					if bytes.Compare(data, resp) != 0 {
						t.Errorf("\nresponse was: %v\nexpected: %v\n",
							resp, data)
					}
				}
			}

		case <-time.After(10 * time.Millisecond):
			t.Errorf("timeout")
		}
	} else {
		t.Errorf("failed to send mtypeEcho msg: %v", err)
	}
}

// TestEchoMsg connects to the server and send an echo message, the server
// side echo handle has a delay inserted to simulate a long running server
// task.  This tests the clients ability to deal with a timeout, the test
// passes if we timeout on waiting for status being signalled via the
// statusChan.
func TestEchoTimeoutMsg(t *testing.T) {
	// Send the "echo" command.
	echoDelay = time.Duration(2 * time.Second)
	data := []byte("echodata")
	datalen := uint16(len(data))
	err := msgClient.WriteTLV(mtypeEcho, datalen, data)

	// Wait 10 milliseconds before declaring an error, this means a timeout
	// will occur since the server side handler has a 2 second delay inserted.
	if err == nil {
		select {
		case sm := <-statusChan:
			if sm.err != nil {
				t.Error(sm.err)
			} else {
				t.Error("unexpected early server handler completion")
			}

		case <-time.After(10 * time.Millisecond):
		}
	} else {
		t.Errorf("failed to send mtypeEcho msg: %v", err)
	}
}
