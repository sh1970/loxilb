package bfd

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	tk "github.com/loxilb-io/loxilib"
)

type SessionState uint8

const (
	BFDAdminDown SessionState = iota
	BFDDown
	BFDInit
	BFDUp
)

const (
	BFDMinSysTXIntervalUs = 100000
	BFDDflSysTXIntervalUs = 200000
	BFDMinSysRXIntervalUs = 200000
)

type WireRaw struct {
	Version       uint8
	Length        uint8
	State         SessionState
	Multi         uint8
	Disc          uint32
	RDisc         uint32
	DesMinTxInt   uint32
	ReqMinRxInt   uint32
	ReqMinEchoInt uint32
}

type Notifer interface {
	BFDSessionNotify(instance string, remote string, state string)
}

type bfdSession struct {
	RemoteName     string
	Instance       string
	Cxn            net.Conn
	State          SessionState
	MyMulti        uint8
	RemMulti       uint8
	MyDisc         uint32
	RemDisc        uint32
	DesMinTxInt    uint32
	RemDesMinTxInt uint32
	ReqMinRxInt    uint32
	TimeOut        uint32
	ReqMinEchoInt  uint32
	LastRxTS       time.Time
	TxTicker       *time.Ticker
	RxTicker       *time.Ticker
	Fin            chan bool
	Mutex          sync.RWMutex
	Notify         Notifer
	PktDat         [24]byte
}

type Struct struct {
	BFDSessMap map[string]*bfdSession
	BFDMtx     sync.RWMutex
}

func StructNew(port uint16) *Struct {
	bfdStruct := new(Struct)

	bfdStruct.BFDSessMap = make(map[string]*bfdSession)
	go bfdStruct.bfdStartListener(port)
	return bfdStruct
}

func (bs *Struct) BFDAddRemote(remoteIP string, port uint16, interval uint32, multi uint8, instance string, cbs Notifer) error {
	bs.BFDMtx.Lock()
	defer bs.BFDMtx.Unlock()

	sess := bs.BFDSessMap[remoteIP]
	if sess != nil {
		return errors.New("bfd existing session")
	}

	if interval < BFDMinSysTXIntervalUs || multi == 0 {
		return errors.New("bfd malformed args")
	}

	sess = new(bfdSession)
	sess.Instance = instance
	sess.Notify = cbs
	err := sess.initialize(remoteIP, port, interval, multi)
	if err != nil {
		return errors.New("bfd failed to init session")
	}

	bs.BFDSessMap[remoteIP] = sess

	return nil
}

func (bs *Struct) BFDDeleteRemote(remoteIP string, port uint16) error {
	bs.BFDMtx.Lock()
	defer bs.BFDMtx.Unlock()

	sess := bs.BFDSessMap[remoteIP]
	if sess == nil {
		return errors.New("no bfd session")
	}

	sess.destruct()
	delete(bs.BFDSessMap, sess.RemoteName)

	bs.BFDSessMap[remoteIP] = sess

	return nil
}

func decodeCtrlPacket(buf []byte, size int) *WireRaw {

	if size < 24 {
		return nil
	}

	var raw WireRaw

	raw.Version = buf[0] >> 5 & 0x7
	raw.State = SessionState(buf[1] >> 6 & 0x3)
	raw.Multi = buf[2]
	raw.Length = buf[3]

	raw.Disc = binary.BigEndian.Uint32(buf[4:])
	raw.RDisc = binary.BigEndian.Uint32(buf[8:])
	raw.DesMinTxInt = binary.BigEndian.Uint32(buf[12:])
	raw.ReqMinRxInt = binary.BigEndian.Uint32(buf[16:])
	raw.ReqMinEchoInt = binary.BigEndian.Uint32(buf[20:])

	return &raw
}

func (bs *Struct) processBFD(conn *net.UDPConn) {
	var buf [1024]byte

	n, _, err := conn.ReadFromUDP(buf[:])
	if err != nil {
		return
	}

	raw := decodeCtrlPacket(buf[:], n)

	remIP := tk.NltoIP(raw.Disc)
	if remIP != nil {
		//fmt.Printf("raw %v:%s:%v\n", raw, remIP.String(), raw.State)
		bs.BFDMtx.Lock()
		defer bs.BFDMtx.Unlock()

		sess := bs.BFDSessMap[remIP.String()]
		if sess != nil {
			sess.RunSessionSM(raw)
		}
	}
}

func (bs *Struct) bfdStartListener(port uint16) error {
	localName := fmt.Sprintf("%s:%d", "0.0.0.0", port)
	addr, err := net.ResolveUDPAddr("udp4", localName)
	if err != nil {
		return errors.New("failed to resolve to BFD addr")
	}

	lc, err1 := net.ListenUDP("udp4", addr)
	if err1 != nil {
		return errors.New("failed to listen to BFD")
	}

	defer lc.Close()

	for {
		bs.processBFD(lc)
	}

}

func (b *bfdSession) RunSessionSM(raw *WireRaw) {
	inst := b.Instance
	rem := b.RemoteName
	oldState := b.State

	b.Mutex.Lock()

	b.RemMulti = raw.Multi
	b.RemDesMinTxInt = raw.DesMinTxInt
	if b.RemDesMinTxInt > b.ReqMinRxInt {
		b.TimeOut = uint32(b.RemMulti) * b.RemDesMinTxInt
	} else {
		b.TimeOut = uint32(b.RemMulti) * b.ReqMinRxInt
	}
	b.LastRxTS = time.Now()

	if raw.State == BFDDown {
		if b.State == BFDDown {
			b.State = BFDInit
			tk.LogIt(tk.LogInfo, "%s: BFD State -> INIT\n", b.RemoteName)
		}
	} else if raw.State == BFDInit {
		if b.State != BFDUp {
			b.State = BFDUp
			tk.LogIt(tk.LogInfo, "%s: BFD State -> UP\n", b.RemoteName)
		}
	} else if raw.State == BFDAdminDown {
		if b.State != BFDAdminDown {
			tk.LogIt(tk.LogInfo, "%s: BFD State -> AdminDown\n", b.RemoteName)
		}
		b.State = BFDAdminDown
	} else if raw.State == BFDUp {
		if b.State != BFDUp {
			tk.LogIt(tk.LogInfo, "%s: BFD State -> UP\n", b.RemoteName)
		}
		b.State = BFDUp
	}
	newState := b.State
	b.Mutex.Unlock()

	b.sendStateNotification(newState, oldState, inst, rem)
}

func (b *bfdSession) checkSessTimeout() {
	inst := b.Instance
	rem := b.RemoteName
	oldState := b.State

	b.Mutex.Lock()
	if b.State == BFDUp {
		if time.Duration(time.Since(b.LastRxTS).Microseconds()) > time.Duration(b.TimeOut) {
			b.State = BFDDown
			tk.LogIt(tk.LogInfo, "%s: BFD State -> Down\n", b.RemoteName)
		}
	}
	newState := b.State
	b.Mutex.Unlock()

	b.sendStateNotification(newState, oldState, inst, rem)
}

func (b *bfdSession) sendStateNotification(newState, oldState SessionState, inst string, remote string) {
	if newState == oldState {
		return
	}

	if newState == BFDUp {
		ciState := "BACKUP"
		if b.MyDisc > b.RemDisc {
			ciState = "MASTER"
		}
		b.Notify.BFDSessionNotify(inst, remote, ciState)
	} else if newState == BFDDown && oldState == BFDUp {
		ciState := "MASTER"
		b.Notify.BFDSessionNotify(inst, remote, ciState)
	} else {
		b.Notify.BFDSessionNotify(inst, remote, "NOT_DEFINED")
	}
}

func (b *bfdSession) bfdSessionTicker() {
	for {
		select {
		case <-b.Fin:
			return
		case t := <-b.RxTicker.C:
			tk.LogIt(-1, "Tick at %v\n", t)
			b.checkSessTimeout()
		case t := <-b.TxTicker.C:
			tk.LogIt(-1, "Tick at %v\n", t)
			b.encodeCtrlPacket()
			b.sendBFDPacket()
		}
	}
}

// getMyDisc - Get My Discriminator based on remote
func getMyDisc(ip net.IP) net.IP {
	// get list of available addresses
	addr, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	var first net.IP

	for _, addr := range addr {
		if ipnet, ok := addr.(*net.IPNet); ok {
			// check if IPv4 or IPv6 is not nil
			if ipnet.IP.To4() != nil || ipnet.IP.To16() != nil {
				if ipnet.Contains(ip) {
					return ipnet.IP
				}
				if first == nil {
					first = ipnet.IP
				}
			}
		}
	}

	return first
}

func (b *bfdSession) initialize(remoteIP string, port uint16, interval uint32, multi uint8) error {
	var err error
	b.RemoteName = fmt.Sprintf("%s:%d", remoteIP, port)

	ip := net.ParseIP(remoteIP)
	if ip == nil {
		return errors.New("address malformed")
	}

	myIP := getMyDisc(ip)
	if myIP == nil {
		return errors.New("my discriminator not found")
	}
	b.MyDisc = tk.IPtonl(myIP)
	b.RemDisc = tk.IPtonl(ip)
	b.MyMulti = multi
	b.DesMinTxInt = interval
	b.ReqMinRxInt = interval
	b.ReqMinEchoInt = interval
	b.State = BFDDown

	b.Cxn, err = net.DialTimeout("udp4", b.RemoteName, 1*time.Second)
	if err != nil || b.Cxn == nil {
		return errors.New("failed to dial BFD")
	}

	b.Fin = make(chan bool)
	b.TxTicker = time.NewTicker(time.Duration(b.DesMinTxInt) * time.Microsecond)
	b.RxTicker = time.NewTicker(time.Duration(BFDMinSysRXIntervalUs) * time.Microsecond)

	go b.bfdSessionTicker()
	return nil
}

func (b *bfdSession) destruct() {
	b.State = BFDAdminDown
	b.Fin <- true
	// Signal ADMIN Down to peer
	b.encodeCtrlPacket()
	b.sendBFDPacket()
}

func (b *bfdSession) encodeCtrlPacket() error {

	b.PktDat[0] = byte(byte(0x1<<5) | byte(0))
	b.PktDat[1] = (uint8(b.State) << 6)
	b.PktDat[2] = b.MyMulti
	b.PktDat[3] = 24

	binary.BigEndian.PutUint32(b.PktDat[4:], uint32(b.MyDisc))
	binary.BigEndian.PutUint32(b.PktDat[8:], uint32(b.RemDisc))
	binary.BigEndian.PutUint32(b.PktDat[12:], uint32(b.DesMinTxInt))
	binary.BigEndian.PutUint32(b.PktDat[16:], uint32(b.ReqMinRxInt))
	binary.BigEndian.PutUint32(b.PktDat[20:], uint32(b.ReqMinEchoInt))

	return nil
}

func (b *bfdSession) sendBFDPacket() error {
	b.Cxn.SetDeadline(time.Now().Add(500 * time.Millisecond))
	_, err := b.Cxn.Write(b.PktDat[:])
	if err != nil {
		tk.LogIt(-1, "Error in sending %s\n", err)
	}
	return err
}
