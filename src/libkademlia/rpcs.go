package libkademlia

// Contains definitions mirroring the Kademlia spec. You will need to stick
// strictly to these to be compatible with the reference implementation and
// other groups' code.

import (
	"net"
	"fmt"
)

type KademliaRPC struct {
	kademlia *Kademlia
}

// Host identification.
type Contact struct {
	NodeID ID
	Host   net.IP
	Port   uint16
}

///////////////////////////////////////////////////////////////////////////////
// PING
///////////////////////////////////////////////////////////////////////////////
type PingMessage struct {
	Sender Contact
	MsgID  ID
}

type PongMessage struct {
	MsgID  ID
	Sender Contact
}

func (k *KademliaRPC) Ping(ping PingMessage, pong *PongMessage) error {
	// TODO: Finish implementation
	pong.MsgID = CopyID(ping.MsgID)
	// Specify the sender
	pong.Sender = k.kademlia.SelfContact
	// Update contact, etc
	// k.kademlia.ContactChan <- &ping.Sender
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// STORE
///////////////////////////////////////////////////////////////////////////////
type StoreRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
	Value  []byte
}

type StoreResult struct {
	MsgID ID
	Err   error
}

func (k *KademliaRPC) Store(req StoreRequest, res *StoreResult) error {
	// TODO: Implement.
	res.MsgID = CopyID(req.MsgID)
	//get the key-value set from request
    newKeyValueSet := KeyValueSet{req.Key, req.Value, make(chan bool),make(chan []byte)}
    fmt.Println("Store : " + req.Key.AsString()+string(req.Value))
    // update hashtable
	k.kademlia.KeyValueChan <- &newKeyValueSet
	// update bucket contact list
	k.kademlia.ContactChan <- &(req.Sender)
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// FIND_NODE
///////////////////////////////////////////////////////////////////////////////
type FindNodeRequest struct {
	Sender Contact
	MsgID  ID
	NodeID ID
}

type FindNodeResult struct {
	MsgID ID
	Nodes []Contact
	Err   error
}

func (k *KademliaRPC) FindNode(req FindNodeRequest, res *FindNodeResult) error {
	// TODO: Implement.
	foundContacts := k.kademlia.FindClosest(req.NodeID, 20)
	res.MsgID = CopyID(req.MsgID)
	res.Nodes = make([]Contact, len(foundContacts))
	res.Nodes = foundContacts
	k.kademlia.ContactChan <- &(req.Sender)
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// FIND_VALUE
///////////////////////////////////////////////////////////////////////////////
type FindValueRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
}

// If Value is nil, it should be ignored, and Nodes means the same as in a
// FindNodeResult.
type FindValueResult struct {
	MsgID ID
	Value []byte
	Nodes []Contact
	Err   error
}

func (k *KademliaRPC) FindValue(req FindValueRequest, res *FindValueResult) error {
	// TODO: Implement.
	res.MsgID = CopyID(req.MsgID)

	// FindRet := new(KeyValueSet)
	// FindRet.Key = req.Key
	// FindRet.KVSearchResChan = make(chan bool)
	// k.KVSearchChan <- FindRet
	// found := <- FindRet.KVSearchResChan
	FindRet, found := k.kademlia.BoolLocalFindValue(req.Key)
	res.Value = make([] byte, len(FindRet.Value))
	if found == true {
		res.Value = FindRet.Value
		return nil
	}
	res.Value = nil
	res.Nodes = k.kademlia.FindClosest(req.Key, 20)
	k.kademlia.ContactChan <- &(req.Sender)
	return nil
}

// For Project 3

type GetVDORequest struct {
	Sender Contact
	VdoID  ID
	MsgID  ID
}

type GetVDOResult struct {
	MsgID ID
	VDO   VanashingDataObject
}

func (k *KademliaRPC) GetVDO(req GetVDORequest, res *GetVDOResult) error {
	// TODO: Implement.
	return nil
}
