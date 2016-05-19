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
	"sort"
)

const (
	Alpha = 3
	B     = 8 * IDBytes
	K     = 20
)

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID				ID
	SelfContact			Contact
	RoutingTable			*Router
	HashTable			map[ID][]byte
	// HashTableKeyChan  chan ID 
	// HashTableValueChan   chan []byte
	ContactChan			chan *Contact
	KeyValueChan			chan *KeyValueSet
	KVSearchChan			chan *KeyValueSet
	BucketsIndexChan		chan int
	BucketResultChan		chan []Contact
}

type Router struct {
	SelfContact			Contact
	Buckets				[][]Contact
}

type KeyValueSet struct {
	Key				ID
	Value				[]byte
	KVSearchBoolChan		chan bool
	KVSearchRestChan		chan []byte
}

type ContactDistance struct {
	contact				Contact
	distance			int
}

type ByDist []ContactDistance
func (d ByDist) Len() int		{ return len(d) }
func (d ByDist) Swap(i, j int)		{ d[i], d[j] = d[j], d[i] }
func (d ByDist) Less(i, j int) bool	{ return d[i].distance < d[j].distance }

func NewKademliaWithId(laddr string, nodeID ID) *Kademlia {
	// TODO: Initialize other state here as you add functionality.
	k := new(Kademlia)
	k.NodeID = nodeID
	k.HashTable = make(map[ID] []byte)
	// k.HashTableKeyChan = make(chan ID)
	// k.HashTableValueChan =make(chan []byte)
	k.ContactChan = make(chan * Contact)
	k.KeyValueChan = make(chan * KeyValueSet)
	k.KVSearchChan = make(chan * KeyValueSet)
	k.BucketsIndexChan = make(chan int)
	k.BucketResultChan = make(chan []Contact)
	k.RoutingTable = new(Router)
	k.RoutingTable.Buckets = make([][]Contact, B)
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
	k.RoutingTable.SelfContact = k.SelfContact
	go handleRequest(k)
	return k
}

func (k *Kademlia) UpdateRoutingTable(contact *Contact){
	// fmt.Println("update started")
	prefixLength := contact.NodeID.Xor(k.NodeID).PrefixLen();
	if prefixLength == B {
		return
	}
	var tmpContact Contact
	found := false
	contactIndex := 0;
	bucket := &k.RoutingTable.Buckets[prefixLength]

	for x, value := range *bucket {
		if value.NodeID.Equals(contact.NodeID){
			tmpContact = value
			contactIndex = x
			found = true
			break
		}
	}
	// fmt.Println("update finished")
	if found == false {
		if len(*bucket) <= K {
			*bucket = append(*bucket, *contact)
		} else {
			_,err:=k.DoPing((*bucket)[0].Host,(*bucket)[0].Port)
			if err == nil  {
				tmpContact := (*bucket)[0]
				*bucket = (*bucket)[1:]
				*bucket = append(*bucket, tmpContact)
			}
		}
	} else {
		*bucket = append((*bucket)[:contactIndex], (*bucket)[(contactIndex+1):]...)
	 	*bucket = append(*bucket, tmpContact)
	}
}

func handleRequest(k *Kademlia) {
	for {
		select {
		case contact := <- k.ContactChan:
			// fmt.Println("get from Contact channel : " + contact.NodeID.AsString())
			k.UpdateRoutingTable(contact)
		case kvset := <- k.KeyValueChan:
			// fmt.Println("get from KeValue channel : " + string(kvset.Value))
			// k.HashTableValueChan <- kvset.Value
			k.HashTable[kvset.Key] = kvset.Value
			// fmt.Println("print Stored value : " + string(k.HashTable[kvset.Key]))
		case kvsearch := <- k.KVSearchChan:
			kvsearch.Value = k.HashTable[kvsearch.Key]
			fmt.Println("print Search value : " + string(k.HashTable[kvsearch.Key]))
			if kvsearch.Value == nil {
				fmt.Println("value not found")
				kvsearch.KVSearchBoolChan <- false
				kvsearch.KVSearchRestChan <- kvsearch.Value
			} else {
				kvsearch.KVSearchBoolChan <- true
				kvsearch.KVSearchRestChan <- kvsearch.Value
			}
		case bucketIndex := <- k.BucketsIndexChan:
			k.BucketResultChan <- k.RoutingTable.Buckets[bucketIndex]
		}
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

	prefixLength := nodeId.Xor(k.RoutingTable.SelfContact.NodeID).PrefixLen()
	k.BucketsIndexChan <- prefixLength
	bucket := <- k.BucketResultChan

	for _, value := range bucket{
		if value.NodeID.Equals(nodeId) {
			k.ContactChan <- &value
			return &value, nil
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

func ConbineHostIP(host net.IP, port uint16) string {
	return host.String() + ":" + strconv.FormatInt(int64(port), 10)
}

func (k *Kademlia) DoPing(host net.IP, port uint16) (*Contact, error) {
	// TODO: Implement
	ping := new(PingMessage)
	ping.Sender = k.SelfContact
	ping.MsgID = NewRandomID()
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
	fmt.Println("receive: " + pong.Sender.NodeID.AsString())

	k.ContactChan <- &(&pong).Sender
	defer client.Close()
	return nil, nil
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

	err = client.Call("KademliaRPC.Store", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
		return &CommandFailed{"Not implemented"}
	}
	defer client.Close()
	return nil
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) ([]Contact, error) {
	// TODO: Implement
	req := FindNodeRequest{k.SelfContact, NewRandomID(), searchKey}
	var res FindNodeResult

	port_str := strconv.Itoa(int(contact.Port))
	client, err := rpc.DialHTTPPath("tcp", ConbineHostIP(contact.Host, contact.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
		return nil, &CommandFailed{"Not implemented"}
	}
	defer client.Close()
	err = client.Call("KademliaRPC.FindNode", req, &res)

	if err != nil {
		log.Fatal("Call: ", err)
		return nil, &CommandFailed{"Not implemented"}
	}
	for i := 0; i < len(res.Nodes); i++ {
		k.ContactChan <- &(res.Nodes[i])
	}
	return res.Nodes, nil
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
		return nil, nil, &CommandFailed{"Not implemented"}
	}
	defer client.Close()
	err = client.Call("KademliaRPC.FindValue", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
		return nil, nil, &CommandFailed{"Not implemented"}
	}

	fmt.Println("---DoFindValue : " + string(res.Value) + "\n")
	for i := 0; i < len(res.Nodes); i++ {
		k.ContactChan <- &(res.Nodes[i])
	}

	return res.Value, res.Nodes, nil
}

func (k *Kademlia) LocalFindValue(searchKey ID) ([]byte, error) {
	// TODO: Implement
	res := new(KeyValueSet)
	res.Key = searchKey
	res.KVSearchBoolChan = make(chan bool)
	res.KVSearchRestChan = make(chan []byte)
	k.KVSearchChan <- res
	found := <- res.KVSearchBoolChan
	Value := <- res.KVSearchRestChan
	if found == true {
		return Value, nil
		fmt.Println("FindValue : " + string(Value) + "\n")
	}
	return []byte(""), nil
}

func (k *Kademlia) BoolLocalFindValue(searchKey ID) (result *KeyValueSet, found bool, Value []byte) {
	result = new(KeyValueSet)
	result.Key = searchKey
	result.KVSearchBoolChan = make(chan bool)
	result.KVSearchRestChan = make(chan []byte)
	k.KVSearchChan <- result
	found = <- result.KVSearchBoolChan
	Value = <- result.KVSearchRestChan
	return result, found, Value
}

func Distance(des ID, bucket []Contact, tempList *[]ContactDistance) {
	for _, value := range bucket {
		desID := value.NodeID.Xor(des)
		dist := desID.ToInt()
		ConDist := &ContactDistance{value, dist}
		*tempList = append(*tempList, *ConDist)
	}
}

func (k *Kademlia) FindClosest(searchID ID, num int) (result []Contact){
	result = make([]Contact, 0)
	tempList := make([]ContactDistance, 0)
	prefixLength := searchID.Xor(k.RoutingTable.SelfContact.NodeID).PrefixLen()
	for i := 0; (prefixLength - i >= 0 || prefixLength + i < B) && len(tempList) < num; i++ {
		if prefixLength == B && prefixLength - i == B {
			tempList = append(tempList, ContactDistance{k.RoutingTable.SelfContact, 0})
			continue
		}
		if prefixLength - i >= 0 {
			k.BucketsIndexChan <- (prefixLength - i)
			bucket := <- k.BucketResultChan
			Distance(searchID, bucket, &tempList)
		}
		if prefixLength + i < B {
			k.BucketsIndexChan <- (prefixLength + i)
			bucket := <- k.BucketResultChan
			Distance(searchID, bucket, &tempList)
		}
	}
	sort.Sort(ByDist(tempList))
	if len(tempList) > num {
		tempList = tempList[:num]
	}

	for _, value := range tempList {
		result = append(result, value.contact)
	}
	return
}

// For project 2!
type IterativeResult struct {
	key					ID
	value				[]byte
	contacts			[]Contact
}

type activeUpdate struct {
	targetID			ID
	boolActive			bool
}

type shortlistUpdate struct {
	BoolSort			bool
	AppendItem			ContactDistance
}

func completed(shortlistGetLenChan chan bool, shortlistResLenChan chan int,
		       shortlistGetContensChan chan bool, shortlistResContensChan chan ContactDistance,
		       activeMapSearchChan chan ID, activeMapResultChan chan bool,
		       closestnode Contact, found_value []byte) bool{
	if found_value != nil {
		return true
	}
	shortlistContentsTmp := make([]ContactDistance, 0)
	shortlistGetLenChan <- true
	shortlistLen := <-shortlistResLenChan
	shortlistGetContensChan <- true
	for ; shortlistLen > 0; shortlistLen-- {
		tmpDistanceContact := <- shortlistResContensChan
		shortlistContentsTmp = append(shortlistContentsTmp, tmpDistanceContact)
	}
	for i := 0; i < len(shortlistContentsTmp) && i < K; i++ {
		activeMapSearchChan <- shortlistContentsTmp[i].contact.NodeID
		activeMapResultBool := <- activeMapResultChan
		if activeMapResultBool == false {
			return false
		}
		fmt.Println(i)
	}
	return true
}

func SendFindNodeQuery(c Contact, activeMapSearchChan chan ID,
	activeMapResultChan chan bool, activeMapUpdateChan chan * activeUpdate,
	waitChan chan int, nodeChan chan Contact) {

	tmpUpdate := new(activeUpdate)
	tmpUpdate.targetID = c.NodeID
	tmpUpdate.boolActive = true
	activeMapUpdateChan <- tmpUpdate        //Set true

	req := FindNodeRequest{c, NewRandomID(), c.NodeID}
	var res FindNodeResult

	port_str := strconv.Itoa(int(c.Port))
	client, err := rpc.DialHTTPPath("tcp", ConbineHostIP(c.Host, c.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
		tmpUpdate.boolActive = false
		activeMapUpdateChan <- tmpUpdate   //Set false
	}
	defer client.Close()
	err = client.Call("KademliaRPC.FindNode", req, &res)

	if err != nil {
		log.Fatal("Call: ", err)
		tmpUpdate.boolActive = false
		activeMapUpdateChan <- tmpUpdate   //Set false
	}
	activeMapSearchChan <- c.NodeID
	activeMapResultBool := <- activeMapResultChan
	if activeMapResultBool == true {
		for _, node := range res.Nodes {
			nodeChan <- node
		}
	}
	waitChan <- 1
	return
}


func SendFindValueQuery(c Contact, activeMapSearchChan chan ID,
	activeMapResultChan chan bool, activeMapUpdateChan chan * activeUpdate,
	waitChan chan int, nodeChan chan Contact, valueChan chan []byte, target ID) {
	tmpUpdate := new(activeUpdate)
	tmpUpdate.targetID = c.NodeID
	tmpUpdate.boolActive = true
	activeMapUpdateChan <- tmpUpdate        //Set true

	req := FindValueRequest{c, NewRandomID(), target}
	res := new(FindValueResult)

	port_str := strconv.Itoa(int(c.Port))
	client, err := rpc.DialHTTPPath("tcp", ConbineHostIP(c.Host, c.Port), rpc.DefaultRPCPath+port_str)
	if err != nil {
		log.Fatal("DialHTTP: ", err)
		tmpUpdate.boolActive = false
		activeMapUpdateChan <- tmpUpdate   //Set false
	}
	defer client.Close()
	err = client.Call("KademliaRPC.FindValue", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
		tmpUpdate.boolActive = false
		activeMapUpdateChan <- tmpUpdate   //Set false
	}

	activeMapSearchChan <- c.NodeID
	activeMapResultBool := <- activeMapResultChan
	if res.Value == nil {
		if activeMapResultBool == true {
			for _, node := range res.Nodes {
				nodeChan <- node
			}
		}
		// fmt.Println(string(res.Value) + "&&-&&&\n")
	} else {
		valueChan <- res.Value
		// fmt.Println(string(res.Value) + "&&&&&\n")
	}

	waitChan <- 1
}

func (k *Kademlia) IterativeFindNode(target ID, findvalue bool) (result *IterativeResult) {
	tempShortList := k.FindClosest(target, K)
	shortlist := make([]ContactDistance, 0)
	shortlistUpdateChan := make(chan * shortlistUpdate)
	shortlistGetLenChan := make(chan bool)
	shortlistResLenChan := make(chan int)
	shortlistGetContensChan := make(chan bool)
	shortlistResContensChan := make(chan ContactDistance)

	var closestNode Contact
	valueChan := make(chan []byte)
	nodeChan := make(chan Contact)
	result = new(IterativeResult)

	visiteMap := make(map[ID]bool)
	activeMap := make(map[ID]bool)
	activeMapSearchChan := make(chan ID)
	activeMapResultChan	:= make(chan bool)
	activeMapUpdateChan := make(chan * activeUpdate)

	if findvalue == true {
		result.key = target
	}
	result.value = nil
	for _, node := range tempShortList {
		shortlist = append(shortlist, ContactDistance{node, node.NodeID.Xor(target).ToInt()})
	}
	go func() {
		for {
			select {
			case tmpUpdate := <-activeMapUpdateChan:
				activeMap[tmpUpdate.targetID] = tmpUpdate.boolActive
			case activeID := <-activeMapSearchChan:
				tmpResult := activeMap[activeID]
				if tmpResult == false {
					activeMapResultChan <- false
				} else {
					activeMapResultChan <- true
				}
			case  shortlistUpdateTmp := <- shortlistUpdateChan:
				if shortlistUpdateTmp.BoolSort == false {
					shortlist = append(shortlist, shortlistUpdateTmp.AppendItem)
				} else {
					sort.Sort(ByDist(shortlist))
				}
			case node := <-nodeChan:
				BoolFound := false
				for _, value := range shortlist {
					if value.contact.NodeID == node.NodeID{
						BoolFound = true
						break
					}
				}
				if BoolFound == false{
					shortlist = append(shortlist, ContactDistance{node, node.NodeID.Xor(target).ToInt()})
				}
			case value := <-valueChan:
				result.value = value
				fmt.Println(string(value) + "Value set in the valueChan\n")
				newNode := new(Contact)
				newNode.NodeID = result.key
				shortlist = append(shortlist, ContactDistance{*newNode, (*newNode).NodeID.Xor(target).ToInt()})
			case getLenBool := <- shortlistGetLenChan:
				if getLenBool == true {
					shortlistResLenChan <- len(shortlist)
				}
			case getContensBool := <- shortlistGetContensChan:
				if getContensBool == true {
					for _, cTmp := range(shortlist) {
						shortlistResContensChan <- cTmp
					}
				}
			}
		}
	}()

	waitChan := make(chan int, Alpha)

	for !completed(shortlistGetLenChan, shortlistResLenChan,
		           shortlistGetContensChan, shortlistResContensChan,
		           activeMapSearchChan, activeMapResultChan,
		           closestNode, result.value) {
		shortlistContents := make([]ContactDistance, 0)
		shortlistGetLenChan <- true
		shortlistLen := <-shortlistResLenChan
		shortlistGetContensChan <- true
		for ; shortlistLen > 0; shortlistLen-- {
			tmpDistanceContact := <- shortlistResContensChan
			shortlistContents = append(shortlistContents, tmpDistanceContact)
		}

		count := 0
		for _, c := range shortlistContents {
			if visiteMap[c.contact.NodeID] == false {
				if count >= Alpha {
					break
				}
				if findvalue == true {
					go SendFindValueQuery(c.contact, activeMapSearchChan, activeMapResultChan,
											activeMapUpdateChan, waitChan, nodeChan, valueChan, target)
				} else {
					go SendFindNodeQuery(c.contact, activeMapSearchChan, activeMapResultChan,
											activeMapUpdateChan, waitChan, nodeChan)
				}
				visiteMap[c.contact.NodeID] = true
				count++
			}
		}
		for ; count > 0; count-- {
			<-waitChan
		}
	}
	result.contacts = make([]Contact, 0)

	shortlistContentsTmp := make([]ContactDistance, 0)
	shortlistGetLenChan <- true
	shortlistLen := <-shortlistResLenChan
	shortlistGetContensChan <- true
	for ; shortlistLen > 0; shortlistLen-- {
		tmpDistanceContact := <- shortlistResContensChan
		shortlistContentsTmp = append(shortlistContentsTmp, tmpDistanceContact)
	}

	sort.Sort(ByDist(shortlistContentsTmp))
	if len(shortlistContentsTmp) > K {
		shortlistContentsTmp = shortlistContentsTmp[:K]
	}
	for _, value := range shortlistContentsTmp {
		result.contacts = append(result.contacts, value.contact)
	}

	return
}

func (k *Kademlia) DoIterativeFindNode(id ID) ([]Contact, error) {
	result := k.IterativeFindNode(id, false)
	if len(result.contacts) > 0 {
		return result.contacts, nil
	} else {
		return result.contacts, nil
	}
	// return nil, &CommandFailed{"Not implemented"}
}

func (k *Kademlia) DoIterativeStore(key ID, value []byte) ([]Contact, error) {
	result := k.IterativeFindNode(key, false)
	for _, c := range result.contacts {
		go k.DoStore(&c, key, value)
	}
	return result.contacts, nil
}

func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {
	result := k.IterativeFindNode(key, true)
	fmt.Println("result.key : " + result.key.AsString() + "\n" + "result.value : " + string(result.value) + "\n")
	if result.value != nil {
		str := "Key: " + result.key.AsString() + " --> Value: " + string(result.value)
		fmt.Println(str + "\n")
		return result.value, nil
	} else {
		fmt.Println("Value is nil\n")
		return nil, nil
	}
	// return nil, &CommandFailed{"Not implemented"}
}

// For project 3!
func (k *Kademlia) Vanish(data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	return
}

func (k *Kademlia) Unvanish(searchKey ID) (data []byte) {
	return nil
}
