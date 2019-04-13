package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
	"log"
	"math"
	"net"
	"os"

	proto "github.com/golang/protobuf/proto"
	msgpb "github.com/iost-official/go-iost/consensus/synchronizer/pb"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
	mplex "github.com/whyrusleeping/go-smux-multiplex"
)

const protocolID = "iostp2p/1.0"

// MessageType represents the message type.
type MessageType uint16
type p2pMessage []byte

// consts.
const (
	_ MessageType = iota
	RoutingTableQuery
	RoutingTableResponse
	NewBlock
	NewBlockHash
	NewBlockRequest
	SyncBlockHashRequest
	SyncBlockHashResponse
	SyncBlockRequest
	SyncBlockResponse
	SyncHeight
	PublishTx

	UrgentMessage = 1
	NormalMessage = 2
)

const (
	chainIDBegin, chainIDEnd           = 0, 4
	messageTypeBegin, messageTypeEnd   = 4, 6
	versionBegin, versionEnd           = 6, 8
	dataLengthBegin, dataLengthEnd     = 8, 12
	dataChecksumBegin, dataChecksumEnd = 12, 16
	reservedBegin, reservedEnd         = 16, 20
	dataBegin                          = 20

	defaultReservedFlag     = 0
	reservedCompressionFlag = 1
)

func newP2PMessage(chainID uint32, messageType MessageType, version uint16, data []byte) *p2pMessage {

	// content := make([]byte, dataBegin+len(data))
	content := make([]byte, dataBegin+len(data))
	binary.BigEndian.PutUint32(content, chainID)
	binary.BigEndian.PutUint16(content[messageTypeBegin:messageTypeEnd], uint16(messageType))
	binary.BigEndian.PutUint16(content[versionBegin:versionEnd], version)
	binary.BigEndian.PutUint32(content[dataLengthBegin:dataLengthEnd], uint32(len(data)))
	binary.BigEndian.PutUint32(content[dataChecksumBegin:dataChecksumEnd], crc32.ChecksumIEEE(data))
	binary.BigEndian.PutUint32(content[reservedBegin:reservedEnd], 0)
	copy(content[dataBegin:], data)

	m := p2pMessage(content)
	return &m
}

func (m *p2pMessage) content() []byte {
	return []byte(*m)
}

func main() {

	//test target /ip4/172.17.0.2/tcp/30000/ipfs/12D3KooWQY9L8PY34smE4btwt7htqkmkMVVFH16t3YP3RY1Q4BuZ

	dest := flag.String("dest", "", "Destination multiaddr string")
	flag.Parse()
	if *dest == "" {
		fmt.Println("Usage:go run test.go -dest=multiaddr")
		os.Exit(1)
	}

	// Creates a new prv key pair for this host.
	prvKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.Fatalln(err)
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:23333")
	if err != nil {
		log.Fatalln(err)
	}

	opts := []libp2p.Option{
		libp2p.Identity(prvKey),
		libp2p.NATPortMap(),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%d", tcpAddr.IP, tcpAddr.Port)),
		libp2p.Muxer(protocolID, mplex.DefaultTransport),
	}
	h, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		log.Fatalln(err)
	}
	// Turn the destination into a multiaddr.
	maddr, err := multiaddr.NewMultiaddr(*dest)
	if err != nil {
		log.Fatalln(err)
	}
	// Extract the peer ID from the multiaddr.
	info, err := peerstore.InfoFromP2pAddr(maddr)
	if err != nil {
		log.Fatalln(err)
	}

	// Add the destination's peer multiaddress in the peerstore.
	// This will be used during connection and stream creation by libp2p.
	h.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	// Start a stream with the destination.
	// Multiaddress of the destination peer is fetched from the peerstore using 'peerId'.
	s, err := h.NewStream(context.Background(), info.ID, protocolID)
	if err != nil {
		log.Fatalln(err)
	}

	//hq := &msgpb.BlockHashQuery{ReqType: 0, Start: 0, End: math.MaxUint32, Nums: nil} //out of memory
	hq := &msgpb.BlockHashQuery{ReqType: 0, Start: 0, End: math.MaxInt64, Nums: nil} //panic: runtime error: makeslice: cap out of range
	bytes, err := proto.Marshal(hq)
	if err != nil {
		log.Fatalln("marshal blockhashquery failed. err=%v", err)
	}
	msg := newP2PMessage(1024, SyncBlockHashRequest, 1, bytes)

	_, err = s.Write(msg.content())
	if err != nil {
		log.Fatalln("writing message failed. err=%v", err)
	}

	for {

	}

}
