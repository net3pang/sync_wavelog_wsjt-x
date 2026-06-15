package udp

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"wavelog-gate/internal/adif"
)

const magic = 0xadbccbda

type QSOEvent struct {
	Call string `json:"call"`
	Band string `json:"band"`
	Mode string `json:"mode"`
	ADIF string `json:"adif"`
}

type Listener struct {
	port      int
	conn      *net.UDPConn
	mu        sync.Mutex
	running   bool
	OnQSO     func(QSOEvent)
	OnLog     func(string)
	logTick   time.Time
	typeCount [256]int
}

func New(port int) *Listener {
	return &Listener{port: port}
}

func (l *Listener) Start() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.running {
		return nil
	}
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%d", l.port))
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	l.conn = conn
	l.running = true
	l.logf("UDP 监听器已启动 (端口 %d)", l.port)
	go l.loop()
	return nil
}

func (l *Listener) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.running = false
	if l.conn != nil {
		l.conn.Close()
		l.conn = nil
	}
}

func (l *Listener) Port() int { return l.port }

func (l *Listener) IsRunning() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.running
}

func (l *Listener) loop() {
	buf := make([]byte, 4096)
	for {
		l.mu.Lock()
		conn := l.conn
		running := l.running
		l.mu.Unlock()
		if !running || conn == nil {
			return
		}
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}
		l.handlePacket(buf[:n])
	}
}

func (l *Listener) logf(f string, args ...interface{}) {
	if l.OnLog != nil {
		l.OnLog(fmt.Sprintf(f, args...))
	}
}

func (l *Listener) handlePacket(data []byte) {
	if len(data) < 8 {
		return
	}
	mg := binary.BigEndian.Uint32(data[0:4])
	typ := binary.BigEndian.Uint32(data[4:8])
	if mg != magic {
		return
	}
	l.typeCount[typ]++
	now := time.Now()
	if now.Sub(l.logTick) >= 10*time.Second {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("UDP 10s 统计"))
		for t := 0; t < len(l.typeCount); t++ {
			if l.typeCount[t] > 0 {
				sb.WriteString(fmt.Sprintf(" | type=%d:%d", t, l.typeCount[t]))
				l.typeCount[t] = 0
			}
		}
		l.logTick = now
		l.logf(sb.String())
	}
	fields := parseStrings(data[8:])
	switch typ {
	case 5:
		l.onLogQSO(fields)
	case 8:
		if len(fields) > 0 {
			l.onADIF(fields[0])
		}
	case 9:
		if len(fields) > 0 {
			l.onADIF(fields[0])
		}
	}
}

func parseStrings(data []byte) []string {
	var out []string
	pos := 0
	for pos < len(data) {
		end := pos
		for end < len(data) && data[end] != 0 {
			end++
		}
		if end > pos {
			out = append(out, string(data[pos:end]))
		} else {
			out = append(out, "")
		}
		pos = end + 1
		for pos < len(data) && data[pos] == 0 {
			pos++
		}
	}
	return out
}

var logFieldNames = []string{
	"id", "date", "time", "dx_call", "dx_grid", "tx_freq", "mode",
	"rst_sent", "rst_rcvd", "tx_power", "comments", "name",
	"my_call", "my_grid", "exchange",
}

func (l *Listener) onLogQSO(fields []string) {
	fmap := make(map[string]string)
	for i, name := range logFieldNames {
		if i < len(fields) {
			fmap[name] = fields[i]
		}
	}
	call := fmap["dx_call"]
	if call == "" {
		return
	}
	date := strings.ReplaceAll(fmap["date"], "-", "")
	tm := strings.ReplaceAll(fmap["time"], ":", "")
	band := adif.FreqToBand(fmap["tx_freq"])
	mode := fmap["mode"]
	rec := adif.MakeRecord(map[string]string{
		"CALL":         call,
		"QSO_DATE":     date,
		"TIME_ON":      tm,
		"BAND":         band,
		"MODE":         mode,
		"RST_SENT":     fmap["rst_sent"],
		"RST_RCVD":     fmap["rst_rcvd"],
		"GRID":         fmap["dx_grid"],
		"NAME":         fmap["name"],
		"COMMENT":      fmap["comments"],
		"QSO_DATE_OFF": date,
		"TIME_OFF":     tm,
	})
	if l.OnQSO != nil {
		l.OnQSO(QSOEvent{Call: call, Band: band, Mode: mode, ADIF: rec})
	}
}

func (l *Listener) onADIF(raw string) {
	call := adif.ExtractField(raw, "CALL")
	if call == "" {
		call = "?"
	}
	if l.OnQSO != nil {
		l.OnQSO(QSOEvent{Call: call, ADIF: raw})
	}
}
