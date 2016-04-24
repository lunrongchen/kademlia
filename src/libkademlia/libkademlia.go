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
	ContactChan		chan *Contact
	KeyValueChan	chan *KeyValueSet
	KVSearchChan	chan *KeyValueSet
}

type Router struct {
	SelfContact 	Contact
	Buckets     	[b]*list.List
}


type KeyValueSet struct {
	Key 			ID
	Value 			[]byte
}

func InitiRoutingTable (RoutingTable *Router) {
 	for i := 0; i < b; i++ {
 		RoutingTable.Buckets[i] = list.New()
 	}
 }


func NewKademliaWithId(laddr string, nodeID ID) *Kademlia {
	// TODO: Initialize other state here as you add functionality.
	k := new(Kademlia)
	k.NodeID = nodeID
    k.HashTable = make(map[ID] []byte)
    // InitiRoutingTable(k.RoutingTable)

    k.ContactChan = make(chan * Contact)
    k.KeyValueChan = make(chan * KeyValueSet)
    k.KVSearchChan = make(chan * KeyValueSet)

	// Set up RPC server
	// NOTE: KademliaRPC is just a wrapper around Kademlia. This type includes
	// the RPC functions.

	s := rpc.NewServer()
	s.Register(&KademliaRPC{k})
	hostname, port, err := net.SplitHostPort(laddr)
	if err != nil {
		return nil
	}
	s.HandleHTTP(rpc.DefaultRPCPath+port,
		rpc.DefaultDebugPath+port)
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

	go handleRequest(k)

	return k
}

func handleRequest(k *Kademlia) {
	for {
		select {
		// case contact := <- k.ContactChan:
			// k.RoutingTable.update(contact)
		case kvset := <-k.KeyValueChan:
			k.HashTable[kvset.Key] = kvset.Value
		// case kvset := <- k.KVSearchChan:
			//todo
		}
		fmt.Sprintf("ii")
	}
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
	// } else {
	// 	recID :=k.SelfContact.NodeID.Xor(nodeId)
	// 	nzero := recID.PrefixLen()
 //        for e := k.RoutingTable.Buckets[b-nzero-1].Front(); e != nil; e = e.Next() {
 //        	if e.Value.NodeID == nodeId {
 //        		k.ContactChan <- &e.Value
 //        		return &e.Value,nil
 //        	}
 //        }
	}
	return nil, &ContactNotFoundError{nodeId, "Not found"}
}

type CommandFailed struct {
	msg string
}

func (e *CommandFailed) Error() string {
	return fmt.Sprintf("%s", e.msg)
}

func ConbineHostIP(host net.IP, port uint16) string {
	return host.String() + ":" + strconv.FormatInt(int64(port), 10)
}

func (k *Kademlia) DoPing(host net.IP, port uint16) (*Contact, error) {
	// TODO: Implement
	ping := PingMessage{k.SelfContact, NewRandomID()}
	var pong PongMessage

	port_str := strconv.Itoa(int(port))
	client, err := rpc.DialHTTPPath("tcp", ConbineHostIP(host, port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	err = client.Call("KademliaCore.Ping", ping, &pong)
	if err != nil {
		log.Fatal("Call: ", err)
		return nil, &CommandFailed{
			"Unable to ping " + fmt.Sprintf("%s:%v", host.String(), port)}
	}
	k.ContactChan <- &(&pong).Sender
	// return "Ping successed : " + pong.MsgID.AsString()
	return nil, &CommandFailed{
		"Ping successed : " + fmt.Sprintf("%s:%v", ConbineHostIP(host, port), pong.MsgID.AsString())}
}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) error {
	// TODO: Implement
	return &CommandFailed{"Not implemented"}
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) ([]Contact, error) {
	// TODO: Implement
	return nil, &CommandFailed{"Not implemented"}
}

func (k *Kademlia) DoFindValue(contact *Contact,
	searchKey ID) (value []byte, contacts []Contact, err error) {
	// TODO: Implement
	return nil, nil, &CommandFailed{"Not implemented"}
}


// func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) error {
// 	// TODO: Implement
// 	hostname, port, err := net.SplitHostPort(firstPeerStr)
// 	client, err := rpc.DialHTTPPath("tcp", firstPeerStr,
// 		rpc.DefaultRPCPath+port)
// 	if err != nil {
// 		log.Fatal("DialHTTP: ", err)
// 	}

// 	log.Printf("Store Key and Value\n")

// 	store := new(StoreRequest)
// 	store.MsgID = NewRandomID()
// 	var storeResult StoreResult
// 	err = client.Call("KademliaRPC.Store",store, &storeResult)
// 	if err != nil {
// 		log.Fatal("Call:", err)
// 		return nil, &CommandFailed{
// 		"Unable to store " + fmt.Sprintf("%s:%v", host.String(), port)}
// 	}

// 	fmt.Printf("Store msgID: %s\n", store.MsgID.AsString())
// }

// func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) ([]Contact, error) {
// 	// TODO: Implement
// 	findNode := new(FindNodeRequest)
// 	findNode.MsgID = NewRandomID()
// 	var findNodeResult FindNodeResult
// 	err = client.Call("KademliaRPC.FindNode",findNode, &findNodeResult)
// 	if err != nil {
// 		log.Fatal("Call:", err)
// 		return nil, &CommandFailed{
// 		"Unable to findNode " + fmt.Sprintf("%s:%v", host.String(), port)}
// 	}
// 	fmt.Printf("Store msgID: %s\n", findNode.MsgID.AsString())
// }

// func (k *Kademlia) DoFindValue(contact *Contact,
// 	searchKey ID) (value []byte, contacts []Contact, err error) {
// 	// TODO: Implement
// 	findValue := new(FindValueRequest)
// 	findValue.MsgID = NewRandomID()
// 	var findValueResult FindValueResult
// 	err = client.Call("KademliaRPC.FindValue",findValue, &findValueResult)
// 	if err != nil {
// 		log.Fatal("Call:", err)
// 		return nil, &CommandFailed{
// 		"Unable to findValue " + fmt.Sprintf("%s:%v", host.String(), port)}
// 	}
// 	fmt.Printf("Store msgID: %s\n", findValue.MsgID.AsString())
// }

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
