package libkademlia

// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"container/list"
)

const (
	alpha = 3
	b     = 8 * IDBytes
	k     = 20
)

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID     		ID
	SelfContact 	Contact
	RoutingTable 	*Router
	HashTable   	map[ID][]byte
}

type Router struct {
	SelfContact 	Contact
	Buckets     	[b]*list.List
}


func InitiRoutingTable (RoutingTable *Router) {
	for i := 0; i < b; i++ {
		RoutingTable.Buckets[i] = list.New()
	}
}
//
func NewKademliaWithId(laddr string, nodeID ID) *Kademlia {
	k := new(Kademlia)
	k.NodeID = nodeID
	//HSQ
    k.HashTable = make(map[ID] []byte)
    InitiRoutingTable(k.RoutingTable)


	// TODO: Initialize other state here as you add functionality.

	// Set up RPC server
	// NOTE: KademliaRPC is just a wrapper around Kademlia. This type includes
	// the RPC functions.

	s := rpc.NewServer()
	s.Register(&KademliaRPC{k})
	hostname, port, err := net.SplitHostPort(laddr)
	if err != nil {
		return nil
	}
	s.HandleHTTP(rpc.DefaultRPCPath+hostname+port,
		rpc.DefaultDebugPath+hostname+port)
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatal("Listen: ", err)
	}

	// Run RPC server forever.
	go http.Serve(l, nil)

	// Add self contact
	hostname, port, _ = net.SplitHostPort(l.Addr().String())
	port_int, _ := strconv.Atoi(port)
	ipAddrStrings, err := net.LookupHost(hostname)
	var host net.IP
	for i := 0; i < len(ipAddrStrings); i++ {
		host = net.ParseIP(ipAddrStrings[i])
		if host.To4() != nil {
			break
		}
	}
	k.SelfContact = Contact{k.NodeID, host, uint16(port_int)}
	return k
}

func NewKademlia(laddr string) *Kademlia {
	return NewKademliaWithId(laddr, NewRandomID())
}

type ContactNotFoundError struct {
	id  ID
	msg string
}

func (e *ContactNotFoundError) Error() string {
	return fmt.Sprintf("%x %s", e.id, e.msg)
}

func (k *Kademlia) FindContact(nodeId ID) (*Contact, error) {
	// TODO: Search through contacts, find specified ID
	// Find contact with provided ID
	if nodeId == k.SelfContact.NodeID {
		return &k.SelfContact, nil
	} else {
		recID :=k.SelfContact.NodeID.Xor(nodeId)
		nzero := recID.PrefixLen()
        for e := k.RoutingTable.Buckets[b-nzero-1].Front(); e != nil; e = e.Next() {
        	if e.Value.NodeID == nodeId {
        		return &e.Value,nil
        	}
        }
	}
	return nil, &ContactNotFoundError{nodeId, "Not found"}
}

type CommandFailed struct {
	msg string
}

func (e *CommandFailed) Error() string {
	return fmt.Sprintf("%s", e.msg)
}

func (k *Kademlia) DoPing(host net.IP, port uint16) (*Contact, error) {
	// TODO: Implement
	ping := new(libkademlia.PingMessage)
	ping.MsgID = libkademlia.NewRandomID()
	var pong libkademlia.PongMessage
	err = client.Call("KademliaRPC.Ping", ping, &pong)
	if err != nil {
		log.Fatal("Call: ", err)
		return nil, &CommandFailed{
		"Unable to ping " + fmt.Sprintf("%s:%v", host.String(), port)
		}
	}
	log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
	log.Printf("pong msgID: %s\n\n", pong.MsgID.AsString())

}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) error {
	// TODO: Implement
	store := new(libkademlia.StoreRequest)
	store.MsgID = libkademlia.NewRandomID()
	var storeResult libkademlia.StoreResult
	err = client.Call("KademliaRPC.Store",store, &storeResult)
	if err != nil {
		log.Fatal("Call:", err)
		return nil, &CommandFailed{
		"Unable to store " + fmt.Sprintf("%s:%v", host.String(), port)
		}
	}
	fmt.Printf("Store msgID: %s\n", store.MsgID.AsString())
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) ([]Contact, error) {
	// TODO: Implement
	findNode := new(libkademlia.FindNodeRequest)
	findNode.MsgID = libkademlia.NewRandomID()
	var findNodeResult libkademlia.FindNodeResult
	err = client.Call("KademliaRPC.FindNode",findNode, &findNodeResult)
	if err != nil {
		log.Fatal("Call:", err)
		return nil, &CommandFailed{
		"Unable to findNode " + fmt.Sprintf("%s:%v", host.String(), port)
		}
	}
	fmt.Printf("Store msgID: %s\n", findNode.MsgID.AsString())
}

func (k *Kademlia) DoFindValue(contact *Contact,
	searchKey ID) (value []byte, contacts []Contact, err error) {
	// TODO: Implement
	findValue := new(libkademlia.FindValueRequest)
	findValue.MsgID = libkademlia.NewRandomID()
	var findValueResult libkademlia.FindValueResult
	err = client.Call("KademliaRPC.FindValue",findValue, &findValueResult)
	if err != nil {
		log.Fatal("Call:", err)
		return nil, &CommandFailed{
		"Unable to findValue " + fmt.Sprintf("%s:%v", host.String(), port)
		}
	}
	fmt.Printf("Store msgID: %s\n", findValue.MsgID.AsString())
}

func (k *Kademlia) LocalFindValue(searchKey ID) ([]byte, error) {
	// TODO: Implement
	for keys := range k.HashTable {
		if keys == searchKey { 
			fmt.Print(k.HashTable[keys])
		}
	}
	return []byte(""), &CommandFailed{"Not implemented"}
}

// For project 2!
func (k *Kademlia) DoIterativeFindNode(id ID) ([]Contact, error) {
	return nil, &CommandFailed{"Not implemented"}
}
func (k *Kademlia) DoIterativeStore(key ID, value []byte) ([]Contact, error) {
	return nil, &CommandFailed{"Not implemented"}
}
func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {
	return nil, &CommandFailed{"Not implemented"}
}

// For project 3!
func (k *Kademlia) Vanish(data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	return
}

func (k *Kademlia) Unvanish(searchKey ID) (data []byte) {
	return nil
}
