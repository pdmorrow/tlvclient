// This package can be used to create clients which can connect to TLV servers
// created with the tlvserv package.
package tlvclient

import (
	"github.com/pdmorrow/tlvserv"
	"log"
	"net"
	"time"
)

// Represents the client construct.
type TLVClient struct {
	conn net.Conn
}

// ReadData() reads datalen bytes using timeout as a read timeout from the
// server.
func (mc *TLVClient) ReadData(datalen uint16,
	timeout time.Duration) ([]byte, error) {
	rb := 0
	tb := int(datalen)
	data := make([]byte, datalen)

	if timeout.Nanoseconds() != 0 {
		// The read operation will eventually timeout.
		mc.conn.SetReadDeadline(time.Now().Add(timeout))
	} else {
		// The read operation will block forever whilst waiting for data.
		mc.conn.SetReadDeadline(time.Time{})
	}

	for rb < tb {
		nbytes, err := mc.conn.Read(data[rb:])
		if err != nil {
			return nil, err
		}

		rb += nbytes
	}

	return data, nil
}

// WriteTLV() write a Type Length Value to the server.
func (mc *TLVClient) WriteTLV(mtype tlvserv.MTypeID,
	datalen uint16,
	data []byte) error {
	packet := make([]byte, datalen+tlvserv.MessageHdrLen)

	packet[0] = uint8(mtype)
	packet[1] = uint8(mtype >> 8)
	packet[2] = uint8(datalen)
	packet[3] = uint8(datalen >> 8)

	if datalen > 0 {
		copy(packet[tlvserv.MessageHdrLen:], data[:datalen])
	}

	_, err := mc.conn.Write(packet)
	return err
}

// Client() create a new client with a connection to a server listening on
// the address specified by the address parameter.  The network parameter
// is used to specify the address family, i.e. tcp4, tcp6.
func Client(network string, address string) (*TLVClient, error) {
	var err error

	mc := new(TLVClient)
	mc.conn, err = net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	// All fine, return the TLVClient object.
	log.Printf("connected to server: %v\n", mc.conn.RemoteAddr())
	return mc, nil
}
