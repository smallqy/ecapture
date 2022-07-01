/*
	Copyright © 2022 CFC4N <cfc4n.cs@gmail.com>
	WebSite: https://www.cnxct.com
*/
package user

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

type AttachType int64

const (
	PROBE_ENTRY AttachType = iota
	PROBE_RET
)

const MAX_DATA_SIZE = 1024 * 4
const SA_DATA_LEN = 14

const (
	SSL2_VERSION    = 0x0002
	SSL3_VERSION    = 0x0300
	TLS1_VERSION    = 0x0301
	TLS1_1_VERSION  = 0x0302
	TLS1_2_VERSION  = 0x0303
	TLS1_3_VERSION  = 0x0304
	DTLS1_VERSION   = 0xFEFF
	DTLS1_2_VERSION = 0xFEFD
)

type tls_version struct {
	version int32
}

func (t tls_version) String() string {
	switch t.version {
	case SSL2_VERSION:
		return "SSL2_VERSION"
	case SSL3_VERSION:
		return "SSL3_VERSION"
	case TLS1_VERSION:
		return "TLS1_VERSION"
	case TLS1_1_VERSION:
		return "TLS1_1_VERSION"
	case TLS1_2_VERSION:
		return "TLS1_2_VERSION"
	case TLS1_3_VERSION:
		return "TLS1_3_VERSION"
	case DTLS1_VERSION:
		return "DTLS1_VERSION"
	case DTLS1_2_VERSION:
		return "DTLS1_2_VERSION"
	}
	return fmt.Sprintf("TLS_VERSION_UNKNOW_%d", t.version)
}

type SSLDataEvent struct {
	module       IModule
	event_type   EVENT_TYPE
	DataType     int64
	Timestamp_ns uint64
	Pid          uint32
	Tid          uint32
	Data         [MAX_DATA_SIZE]byte
	Data_len     int32
	Comm         [16]byte
	Fd           uint32
	Version      int32
}

func (this *SSLDataEvent) Decode(payload []byte) (err error) {
	buf := bytes.NewBuffer(payload)
	if err = binary.Read(buf, binary.LittleEndian, &this.DataType); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Timestamp_ns); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Pid); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Tid); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Data); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Data_len); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Comm); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Fd); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Version); err != nil {
		return
	}

	return nil
}

func (this *SSLDataEvent) GetUUID() string {
	return fmt.Sprintf("%d_%d_%s_%d_%d", this.Pid, this.Tid, CToGoString(this.Comm[:]), this.Fd, this.DataType)
}

func (this *SSLDataEvent) Payload() []byte {
	return this.Data[:this.Data_len]
}

func (this *SSLDataEvent) PayloadLen() int {
	return int(this.Data_len)
}

func (this *SSLDataEvent) StringHex() string {
	addr := this.module.(*MOpenSSLProbe).GetConn(this.Pid, this.Fd)

	var perfix, connInfo string
	switch AttachType(this.DataType) {
	case PROBE_ENTRY:
		connInfo = fmt.Sprintf("%sRecived %d%s bytes from %s%s%s", COLORGREEN, this.Data_len, COLORRESET, COLORYELLOW, addr, COLORRESET)
		perfix = COLORGREEN
	case PROBE_RET:
		connInfo = fmt.Sprintf("%sSend %d%s bytes to %s%s%s", COLORPURPLE, this.Data_len, COLORRESET, COLORYELLOW, addr, COLORRESET)
		perfix = fmt.Sprintf("%s\t", COLORPURPLE)
	default:
		perfix = fmt.Sprintf("UNKNOW_%d", this.DataType)
	}

	b := dumpByteSlice(this.Data[:this.Data_len], perfix)
	b.WriteString(COLORRESET)

	v := tls_version{version: this.Version}
	s := fmt.Sprintf("PID:%d, Comm:%s, TID:%d, %s, Version:%s, Payload:\n%s", this.Pid, CToGoString(this.Comm[:]), this.Tid, connInfo, v.String(), b.String())
	return s
}

func (this *SSLDataEvent) String() string {
	addr := this.module.(*MOpenSSLProbe).GetConn(this.Pid, this.Fd)

	var perfix, connInfo string
	switch AttachType(this.DataType) {
	case PROBE_ENTRY:
		connInfo = fmt.Sprintf("%sRecived %d%s bytes from %s%s%s", COLORGREEN, this.Data_len, COLORRESET, COLORYELLOW, addr, COLORRESET)
		perfix = COLORGREEN
	case PROBE_RET:
		connInfo = fmt.Sprintf("%sSend %d%s bytes to %s%s%s", COLORPURPLE, this.Data_len, COLORRESET, COLORYELLOW, addr, COLORRESET)
		perfix = COLORPURPLE
	default:
		connInfo = fmt.Sprintf("%sUNKNOW_%d%s", COLORRED, this.DataType, COLORRESET)
	}
	v := tls_version{version: this.Version}
	s := fmt.Sprintf("PID:%d, Comm:%s, TID:%d, Version:%s, %s, Payload:\n%s%s%s", this.Pid, bytes.TrimSpace(this.Comm[:]), this.Tid, v.String(), connInfo, perfix, string(this.Data[:this.Data_len]), COLORRESET)
	return s
}

func (this *SSLDataEvent) SetModule(module IModule) {
	this.module = module
}

func (this *SSLDataEvent) Module() IModule {
	return this.module
}

func (this *SSLDataEvent) Clone() IEventStruct {
	event := new(SSLDataEvent)
	event.module = this.module
	event.event_type = EVENT_TYPE_OUTPUT
	return event
}

func (this *SSLDataEvent) EventType() EVENT_TYPE {
	return this.event_type
}

//  connect_events map
/*
uint64_t timestamp_ns;
  uint32_t pid;
  uint32_t tid;
  uint32_t fd;
  char sa_data[SA_DATA_LEN];
  char comm[TASK_COMM_LEN];
*/
type ConnDataEvent struct {
	module      IModule
	event_type  EVENT_TYPE
	TimestampNs uint64
	Pid         uint32
	Tid         uint32
	Fd          uint32
	SaData      [SA_DATA_LEN]byte
	Comm        [16]byte
	Addr        string
}

func (this *ConnDataEvent) Decode(payload []byte) (err error) {
	buf := bytes.NewBuffer(payload)
	if err = binary.Read(buf, binary.LittleEndian, &this.TimestampNs); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Pid); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Tid); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Fd); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.SaData); err != nil {
		return
	}
	if err = binary.Read(buf, binary.LittleEndian, &this.Comm); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(this.SaData[0:2])
	ip := net.IPv4(this.SaData[2], this.SaData[3], this.SaData[4], this.SaData[5])
	this.Addr = fmt.Sprintf("%s:%d", ip, port)
	return nil
}

func (this *ConnDataEvent) StringHex() string {
	s := fmt.Sprintf("PID:%d, Comm:%s, TID:%d, FD:%d, Addr: %s", this.Pid, bytes.TrimSpace(this.Comm[:]), this.Tid, this.Fd, this.Addr)
	return s
}

func (this *ConnDataEvent) String() string {
	s := fmt.Sprintf("PID:%d, Comm:%s, TID:%d, FD:%d, Addr: %s", this.Pid, bytes.TrimSpace(this.Comm[:]), this.Tid, this.Fd, this.Addr)
	return s
}

func (this *ConnDataEvent) SetModule(module IModule) {
	this.module = module
}

func (this *ConnDataEvent) Module() IModule {
	return this.module
}

func (this *ConnDataEvent) Clone() IEventStruct {
	event := new(ConnDataEvent)
	event.module = this.module
	event.event_type = EVENT_TYPE_MODULE_DATA
	return event
}

func (this *ConnDataEvent) EventType() EVENT_TYPE {
	return this.event_type
}

func (this *ConnDataEvent) GetUUID() string {
	return fmt.Sprintf("%d_%d_%s_%d", this.Pid, this.Tid, bytes.TrimSpace(this.Comm[:]), this.Fd)
}

func (this *ConnDataEvent) Payload() []byte {
	return []byte(this.Addr)
}

func (this *ConnDataEvent) PayloadLen() int {
	return len(this.Addr)
}
