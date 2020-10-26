package trace

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Protocol constants
const (
	protocolICMP     = 1
	protocolIPv6ICMP = 58
)

var (
	errLocalAddrNotFound = errors.New("Local addr not found")
)

// Tracer struct
type Tracer struct {
	interval     time.Duration
	localAddr    string
	localIPAddr  *net.IPAddr
	targetAddr   string
	targetIPAddr *net.IPAddr
	conn         *icmp.PacketConn
	body         *icmp.Echo
	message      *icmp.Message
	binMessage   *[]byte
	// channel for stopping goroutines
	signalChannel chan os.Signal
	// all ip addresses
	path []net.IP
}

// Host struct contain information about each host
type Host struct {
	Addr   string
	IPAddr *net.IPAddr
}

// NewTracer - create new tracer struct
func NewTracer(targetAddr string) (*Tracer, error) {
	tracer := &Tracer{}
	if err := tracer.setMessage(); err != nil {
		return nil, err
	}
	if err := tracer.setLocalAddr(); err != nil {
		return nil, err
	}
	if err := tracer.setTargetAddr(targetAddr); err != nil {
		return nil, err
	}
	tracer.signalChannel = make(chan os.Signal, 1)
	signal.Notify(tracer.signalChannel, os.Interrupt)
	tracer.interval = time.Second
	return tracer, nil
}

// set default message
func (tracer *Tracer) setMessage() error {
	tracer.body = &icmp.Echo{
		ID:   os.Getpid() & 0xffff,
		Seq:  1,
		Data: []byte("DEFAULT-MESSAGE"),
	}
	tracer.message = &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: tracer.body,
	}
	binMessage, err := tracer.message.Marshal(nil)
	if err != nil {
		return err
	}
	tracer.binMessage = &binMessage
	return nil
}

// Start - start trace
func (tracer *Tracer) Start() error {
	fmt.Printf("traceroute to %s (%v)\n\n", tracer.targetAddr, tracer.targetIPAddr)
	go tracer.catchExitSignal()
	conn, err := icmp.ListenPacket("ip4:icmp", tracer.localIPAddr.String())
	if err != nil {
		return err
	}
	tracer.conn = conn
	defer tracer.conn.Close()
	go tracer.receivePackets()
	if err := tracer.sendPackets(); err != nil {
		return err
	}
	return nil
}

// catch Ctrl+C signal and close program with printing stat
func (tracer *Tracer) catchExitSignal() {
	switch <-tracer.signalChannel {
	case os.Interrupt:
		// print some statistics
		fmt.Printf("\nPath length = %d: (%v)", len(tracer.path), tracer.localIPAddr)
		for i := range tracer.path {
			fmt.Printf(" -> (%v)", tracer.path[i])
		}
		fmt.Printf("\nexit\n")
		os.Exit(0)
	}
}

// send with ttl++
func (tracer *Tracer) sendPackets() error {
	ttl := 1
	for {
		select {
		case <-tracer.signalChannel:
			return nil
		default:
			tracer.conn.IPv4PacketConn().SetTTL(ttl)
			tracer.conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)
			if _, err := tracer.conn.WriteTo(*tracer.binMessage, tracer.targetIPAddr); err != nil {
				return err
			}
			time.Sleep(tracer.interval)
		}
		ttl++
	}
}

// wait for packet from target host
func (tracer *Tracer) receivePackets() error {
	buffer := make([]byte, 512)
	packetCount := 0
	for {
		select {
		case <-tracer.signalChannel:
			return nil
		default:
			_, controlMessage, peer, err := tracer.conn.IPv4PacketConn().ReadFrom(buffer)
			if err != nil {
				return err
			}
			packetCount++
			tracer.path = append(tracer.path, controlMessage.Src)
			if controlMessage.Src.String() == tracer.targetIPAddr.IP.String() {
				tracer.signalChannel <- os.Interrupt
			}
			fmt.Printf("%d. Packet from (%v)\n\n", packetCount, peer)
		}
	}
}

func (tracer *Tracer) setTargetAddr(targetAddr string) error {
	tracer.targetAddr = targetAddr
	addr, err := net.ResolveIPAddr("ip", targetAddr)
	if err != nil {
		return err
	}
	tracer.targetIPAddr = addr
	return nil
}

// setLocalAddr ...
func (tracer *Tracer) setLocalAddr() error {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return err
	}
	for _, i := range netInterfaces {
		if strings.Contains(i.Flags.String(), "up") &&
			strings.Contains(i.Flags.String(), "broadcast") &&
			strings.Contains(i.Flags.String(), "multicast") {
			ipaddrs, err := i.Addrs()
			if err != nil {
				return err
			}
			for _, addr := range ipaddrs {
				switch v := addr.(type) {
				case *net.IPNet:
					ip := v.IP
					tracer.localIPAddr = &net.IPAddr{
						IP:   ip,
						Zone: "",
					}
					return nil
				case *net.IPAddr:
					//ip := v.IP
					//fmt.Println(ip)
				default:
				}
			}
			return nil
		}
	}
	return errLocalAddrNotFound
}
