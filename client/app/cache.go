package app

import (
	"encoding/json"
	"github.com/scjtqs2/p2p_rdp/common"
	"math/rand"
	"net"
	"strconv"
	"time"
)

//var cacheLock sync.Mutex

// 生成seq
func makeSeq() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Seed(time.Now().UnixNano())
	return strconv.Itoa(r.Intn(1000000))
}

func (l *UdpListener) setSeq(seq string) error {
	return l.Cache.SetWithExpire(seq, "1", 10*time.Second)
}

func (l *UdpListener) checkSeq(seq string) bool {
	check, err := l.Cache.Get(seq)
	if err != nil {
		return false
	}
	return check == "1"
}

func (l *UdpListener) returnSeq(seq string, resturnAddr *net.UDPAddr) {
	reSeq, _ := json.Marshal(&common.UDPMsg{Code: common.UDP_TYPE_SEQ_RESPONSE, Data: []byte(seq), Seq: seq})
	l.LocalConn.WriteToUDP(reSeq, resturnAddr)
}
