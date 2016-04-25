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
	KVSearchChan	chan *KeyValueSetSearch
    FindNodeChan	chan *FNodeChan
}

type FNodeChan struct {
	GetContactChan	chan *Contact
	NodeID 	 		ID
}

type Router struct {
	SelfContact 	Contact
	Buckets     	[b]*list.List
}


type KeyValueSet struct {
	Key 			ID
	Value 			[]byte
}

type KeyValueSetSearch struct {
	Key 			ID
	Value 			chan []byte
}

func InitiRoutingTable (k *Kademlia) {
 	for i := 0; i < b; i++ {
 		k.RoutingTable.Buckets[i] = list.New()
 	}
 	k.RoutingTable.SelfContact = k.SelfContact
 }


func NewKademliaWithId(laddr string, nodeID ID) *Kademlia {
	// TODO: Initialize other state here as you add functionality.
	k := new(Kademlia)
	k.NodeID = nodeID
    k.HashTable = make(map[ID] []byte)
    k.RoutingTable = new(Router)
    InitiRoutingTable(k)

    k.ContactChan = make(chan * Contact)
    k.KeyValueChan = make(chan * KeyValueSet)
    k.KVSearchChan = make(chan * KeyValueSetSearch)
    k.FindNodeChan = make(chan *FNodeChan)

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

func (k *Kademlia) UpdateRoutingTable (contact *Contact){
	prefixLength := contact.NodeID.Xor(k.NodeID).PrefixLen();
	bucket := k.RoutingTable.Buckets[prefixLength]
	for e := bucket.Front(); e != nil; e = e.Next(){
		if contact.NodeID == e.Value.(Contact).NodeID {
			bucket.MoveToBack(e)
			return
		}else{
			if bucket.Len() <= 20 {
				bucket.PushBack(contact)
				}else{
				  _, err :=	k.DoPing(bucket.Front().Value.(Contact).Host, bucket.Front().Value.(Contact).Port) 
					if err != nil {
						bucket.Remove(e)
						bucket.PushBack(contact)
					} else {
						break
					}
				}
			}
	}
}

func handleRequest(k *Kademlia) {
	for {
		select {
	    case contact := <- k.ContactChan:
			k.UpdateRoutingTable(contact)
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
	err = client.Call("KademliaRPC.Ping", ping, &pong)
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
	req := StoreRequest{k.SelfContact, NewRandomID(), key, value}
	var res StoreResult

	port_str := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", ConbineHostIP(contact.Host, contact.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	defer client.Close()
	err = client.Call("KademliaRPC.Store", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
		return &CommandFailed{"Not implemented"}
	}
	return &CommandFailed{"Store implemented"}
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) ([]Contact, error) {
	// TODO: Implement
	req := FindNodeRequest{k.SelfContact, NewRandomID(), searchKey}
	var res FindNodeResult

	port_str := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", ConbineHostIP(contact.Host, contact.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	defer client.Close()
	err = client.Call("KademliaRPC.FindNode", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
		return nil, &CommandFailed{"Not implemented"}

	}
	return nil, &CommandFailed{"FindNode implemented"}
}

func (k *Kademlia) DoFindValue(contact *Contact,
	searchKey ID) (value []byte, contacts []Contact, err error) {
	// TODO: Implement
	req := FindValueRequest{*contact, NewRandomID(), searchKey}
	res := new(FindValueResult)

	port_str := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", ConbineHostIP(contact.Host, contact.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	defer client.Close()
	err = client.Call("KademliaRPC.FindValue", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
		return nil, nil, &CommandFailed{"Not implemented"}
	}
	return nil, nil, &CommandFailed{"FindValue implemented"}
}



func (k *Kademlia) LocalFindValue(searchKey ID) ([]byte, error) {
	// TODO: Implement
	valueChan := make(chan []byte)
	KVSet := KeyValueSetSearch{searchKey,valueChan}
	k.KVSearchChan <- &KVSet
	value := <- valueChan
	if value != nil {
		return value,&CommandFailed{"FindLocalValue implemented"}
	} else {
	return nil, &CommandFailed{"Not implemented"}
}
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
