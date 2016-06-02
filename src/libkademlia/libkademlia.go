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
	"time"
)

const (
	Alpha = 3
	B     = 8 * IDBytes
	K     = 20
	LimitedTime = 300 * time.Millisecond
)

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID				ID
	SelfContact			Contact
	RoutingTable		*Router
	HashTable			map[ID][]byte
	VDOHashTable		map[ID] VanashingDataObject
	ContactChan			chan *Contact
	KeyValueChan			chan *KeyValueSet
	KVSearchChan			chan *KeyValueSet
	BucketsIndexChan		chan int
	BucketResultChan		chan []Contact
	VDOchan					chan VDOpair
	getVDOchan				chan getVDO
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

type VDOpair struct {
	Key 		ID
	VDO 		VanashingDataObject
}

type getVDO struct {
	Key 				ID
	VDOresultchan 		chan VanashingDataObject
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
	k.ContactChan = make(chan * Contact)
	k.KeyValueChan = make(chan * KeyValueSet)
	k.KVSearchChan = make(chan * KeyValueSet)
	k.BucketsIndexChan = make(chan int)
	k.BucketResultChan = make(chan []Contact)
	k.RoutingTable = new(Router)
	k.RoutingTable.Buckets = make([][]Contact, B)
	k.VDOHashTable = make(map[ID] VanashingDataObject)
	k.VDOchan = make(chan VDOpair)
	k.getVDOchan = make(chan getVDO)
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
	// fmt.Println("bucket update finished")
}

func handleRequest(k *Kademlia) {
	for {
		select {
		case contact := <- k.ContactChan: 
			// fmt.Println("get from Contact channel : " + contact.NodeID.AsString())
			k.UpdateRoutingTable(contact)
		case kvset := <- k.KeyValueChan:
			// fmt.Println("get from KeValue channel : " + string(kvset.Value))
			k.HashTable[kvset.Key] = kvset.Value
			// fmt.Println("print Stored value : " + string(k.HashTable[kvset.Key]))
		case kvset := <- k.KVSearchChan:
			kvset.Value = k.HashTable[kvset.Key]
			// fmt.Println("print Search value : " + string(k.HashTable[kvset.Key]))
			if kvset.Value == nil {
				// fmt.Println("value not found")
				kvset.KVSearchBoolChan <- false
				kvset.KVSearchRestChan <- kvset.Value
			} else {
				kvset.KVSearchBoolChan <- true
				kvset.KVSearchRestChan <- kvset.Value
			}
		case bucketIndex := <- k.BucketsIndexChan:
			k.BucketResultChan <- k.RoutingTable.Buckets[bucketIndex]
		case vdostore := <- k.VDOchan:
			k.VDOHashTable[vdostore.Key] = vdostore.VDO
		case getkey := <- k.getVDOchan:
			getvdo := k.VDOHashTable[getkey.Key] 
			getkey.VDOresultchan <- getvdo

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
	return &(&pong).Sender, nil
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
		if prefixLength + i < B && i != 0{
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

func completed(shortlistGetLenChan chan bool, shortlistResLenChan chan int, shortlistGetContensChan chan bool, 
	shortlistResContensChan chan ContactDistance, activeMapSearchChan chan ID, activeMapResultChan chan bool, 
	closestnode Contact, valueSearchChan chan bool, valueResultChan chan bool) bool{
	valueSearchChan <- true
	valueSearchResult := <- valueResultChan
	if valueSearchResult == true {
		return true
	}

// func completed(shortlistGetLenChan chan bool, shortlistResLenChan chan int, shortlistGetContensChan chan bool, 
// 	shortlistResContensChan chan ContactDistance, activeMapSearchChan chan ID, activeMapResultChan chan bool, 
// 	closestnode Contact) bool{

// func completed(shortlistGetLenChan chan bool, shortlistResLenChan chan int, shortlistGetContensChan chan bool, 
// 	shortlistResContensChan chan ContactDistance, activeMapSearchChan chan ID, activeMapResultChan chan bool, 
// 	closestnode Contact, found_value []byte) bool{
// 	if found_value != nil {
// 		return true
// 	}
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
	}
	return true
}

func (k *Kademlia) SendFindNodeQuery(c Contact, activeMapSearchChan chan ID, 
	activeMapResultChan chan bool, activeMapUpdateChan chan * activeUpdate, 
	waitChan chan int, nodeChan chan Contact,target ID) {
	
	tmpUpdate := new(activeUpdate)
	tmpUpdate.targetID = c.NodeID
	tmpUpdate.boolActive = true
	activeMapUpdateChan <- tmpUpdate        
	res,err := k.DoFindNode(&c, target)
	if err != nil {
		tmpUpdate.boolActive = false
	}
	activeMapSearchChan <- c.NodeID
	activeMapResultBool := <- activeMapResultChan
	if activeMapResultBool == true {
		for _, node := range res {
			nodeChan <- node
		}
	}
	waitChan <- 1
	return
}


func (k *Kademlia) SendFindValueQuery(c Contact, activeMapSearchChan chan ID, 
	activeMapResultChan chan bool, activeMapUpdateChan chan * activeUpdate, 
	waitChan chan int, nodeChan chan Contact, valueChan chan []byte, target ID) {
	tmpUpdate := new(activeUpdate)
	tmpUpdate.targetID = c.NodeID
	tmpUpdate.boolActive = true
	activeMapUpdateChan <- tmpUpdate        

	value,res,err := k.DoFindValue(&c, target)
	if err != nil {
		tmpUpdate.boolActive = false
		activeMapUpdateChan <- tmpUpdate
	}
	activeMapSearchChan <- c.NodeID
	activeMapResultBool := <- activeMapResultChan
	if value == nil {
		if activeMapResultBool == true {
			for _, node := range res {
				nodeChan <- node
			}
		}
	} else {
		valueChan <- value
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

	valueSearchChan := make(chan bool)
	valueResultChan := make(chan bool)

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
				// fmt.Println(string(value) + "Value set in the valueChan\n")
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
			case valueSearchTmp := <- valueSearchChan:
				if valueSearchTmp == true {
					if result.value != nil {
						valueResultChan <- true						
					} else {
						valueResultChan <- false
					}
				} 
			}
		}
	}()
	
	waitChan := make(chan int, Alpha)

	for !completed(shortlistGetLenChan, shortlistResLenChan, 
		           shortlistGetContensChan, shortlistResContensChan, 
		           activeMapSearchChan, activeMapResultChan, 
		           // closestNode) {
		           closestNode, valueSearchChan, valueResultChan) {
				   // closestNode, result.value) {
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
					go k.SendFindValueQuery(c.contact, activeMapSearchChan, activeMapResultChan, 
											activeMapUpdateChan, waitChan, nodeChan, valueChan, target)
				} else {
					go k.SendFindNodeQuery(c.contact, activeMapSearchChan, activeMapResultChan, 
											activeMapUpdateChan, waitChan, nodeChan, target)
				}
				visiteMap[c.contact.NodeID] = true
				count++
			}
		}
		timeOut := make(chan bool, 1)
		go func () {
			time.Sleep(LimitedTime)
			timeOut <- true
		}()
		break_for_loop := false

		for ; count > 0; count-- {
			select {
			case boolTimeOut := <- timeOut:
				fmt.Println("timeout")
				fmt.Println("Before Break")
				if boolTimeOut == true {
					break_for_loop = true
				}
			case <-waitChan:
				// break
			}
			if break_for_loop == true {
				break
			}
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
}

func (k *Kademlia) DoIterativeStore(key ID, value []byte) ([]Contact, error) {
	result := k.IterativeFindNode(key, false)
	for _, c := range result.contacts {
		 k.DoStore(&c, key, value)
	}
	return result.contacts, nil
}

func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {
	result := k.IterativeFindNode(key, true)
	fmt.Println("result.key : " + result.key.AsString() + "\n"  +"\n")
	if result.value != nil {
		str := "Key: " + result.key.AsString()  
		fmt.Println(str + "\n")
		return result.value, nil
	} else {
		fmt.Println("Find Value is nil\n")
		return nil, nil
	}
}

// For project 3!
func (k *Kademlia) Vanish(vdoID ID, data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	vdo = k.VanishData(data, numberKeys, threshold, timeoutSeconds)
	VDOstore := VDOpair {vdoID, vdo}
	k.VDOchan <- VDOstore
	return vdo
}
// func (k *Kademlia) Unvanish(nodeID ID, vdoID ID) (data []byte) {
// 		_,_ = k.DoIterativeFindNode(nodeID)
// 		return nil
// }
func (k *Kademlia) Unvanish(nodeID ID, vdoID ID) (data []byte) {
	if nodeID.Compare(k.NodeID) == 0 {
		VDOrequest := getVDO {vdoID, make(chan VanashingDataObject)}
		k.getVDOchan <- VDOrequest
		vdo := <- VDOrequest.VDOresultchan
		data = k.UnvanishData(vdo)
		return data
	} else {
		req := new(GetVDORequest)
		contacts,_ := k.DoIterativeFindNode(nodeID)
		for _,c := range contacts {
			if c.NodeID == nodeID {
				req.Sender = c
				break
			}
		}
		if req.Sender.NodeID != nodeID {
			return nil
		}
		var res GetVDOResult
		req.MsgID = NewRandomID()
		req.VdoID = vdoID
		port_str := strconv.Itoa(int(k.SelfContact.Port))
		client, err := rpc.DialHTTPPath("tcp", ConbineHostIP(k.SelfContact.Host, k.SelfContact.Port), rpc.DefaultRPCPath+port_str)
		if err != nil {
			log.Fatal("DialHTTP: ", err)
		}

		err = client.Call("KademliaRPC.GetVDO", req, &res)
		if err != nil {
			log.Fatal("Call: ", err)
			return nil
		}
		data = k.UnvanishData(res.VDO)
		if data != nil{
			return data
		} else {
			return nil
		}
	}
}