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
	"math"
)

const (
	alpha = 3
	b     = 8 * IDBytes
	k     = 20
)

// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID				ID
	SelfContact			Contact
	RoutingTable			*Router
	HashTable			map[ID][]byte
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
	k.ContactChan = make(chan * Contact)
	k.KeyValueChan = make(chan * KeyValueSet)
	k.KVSearchChan = make(chan * KeyValueSet)
	k.BucketsIndexChan = make(chan int)
	k.BucketResultChan = make(chan []Contact)
	k.RoutingTable = new(Router)
	k.RoutingTable.Buckets = make([][]Contact, b)
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
	if prefixLength == 160 {
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
		if len(*bucket) <= 20 {
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
			// fmt.Println("get from channel from ContantChan: " + contact.NodeID.AsString())
			k.UpdateRoutingTable(contact)
		case kvset := <- k.KeyValueChan:
			fmt.Println("get from channel : " + string(kvset.Value))
			k.HashTable[kvset.Key] = kvset.Value
			fmt.Println("print value : " + string(k.HashTable[kvset.Key]))
		case kvset := <- k.KVSearchChan:
			kvset.Value = k.HashTable[kvset.Key]
			if kvset.Value == nil {
				kvset.KVSearchBoolChan <- false
				kvset.KVSearchRestChan <- kvset.Value
			} else {
				kvset.KVSearchBoolChan <- true
				kvset.KVSearchRestChan <- kvset.Value
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
	// fmt.Println("receive: " + pong.Sender.NodeID.AsString())

	k.ContactChan <- &(&pong).Sender
	defer client.Close()
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
	}
	defer client.Close()
	err = client.Call("KademliaRPC.FindNode", req, &res)

	if err != nil {
		log.Fatal("Call: ", err)
		return nil, &CommandFailed{"Not implemented"}

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
	}
	defer client.Close()
	err = client.Call("KademliaRPC.FindValue", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
		return nil, nil, &CommandFailed{"Not implemented"}
	}
	return res.Value, res.Nodes, &CommandFailed{"FindValue implemented"}
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
	return []byte(""), &CommandFailed{"Not implemented"}
}

func (k *Kademlia) BoolLocalFindValue(searchKey ID) (result *KeyValueSet, found bool, Value []byte) {
	result = new(KeyValueSet)
	result.Key = searchKey
	result.KVSearchBoolChan = make(chan bool)
	result.KVSearchRestChan = make(chan []byte)
	k.KVSearchChan <- result
	found = <- result.KVSearchBoolChan
	Value = <- result.KVSearchRestChan
	return
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
	for i := 0; (prefixLength - i >= 0 || prefixLength + i < IDBytes) && len(tempList) < num; i++ {
		if prefixLength == IDBytes && prefixLength - i == IDBytes {
			tempList = append(tempList, ContactDistance{k.RoutingTable.SelfContact, 0})
			continue
		}
		if prefixLength - i >= 0 {
			k.BucketsIndexChan <- (prefixLength - i)
			bucket := <- k.BucketResultChan
			Distance(searchID, bucket, &tempList)
		}
		if prefixLength + i < IDBytes {
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


func (k *Kademlia) FindNodeQuery(contact *Contact, searchKey ID) ([]Contact, *Contact, error) {
	result,err:=k.DoFindNode(contact ,searchKey)
	return result,contact,err
}

func (k *Kademlia) FindValueQuery(contact *Contact, searchKey ID) ([]byte,[]Contact, *Contact, error) {
	foundvalue,contactresult,err:=k.DoFindValue(contact,searchKey)
	return foundvalue,contactresult,contact,err
}

func (k *Kademlia) DoIterativeFindNode(id ID) ([]Contact, error) {
	tempShortList := k.FindClosest(id, 20)
	shortlist := make([]Contact, 0)
	unactivelist := make([]ContactDistance,0)

	unactivelistchan := make(chan []Contact)
    sendrequestchan := make(chan bool)
	readlistchan := make(chan []ContactDistance)
	shortlistchan := make(chan *Contact)
	countchan := make(chan bool)
	resultchan := make(chan []Contact)
	for _, node := range tempShortList {
		unactivelist = append(unactivelist, ContactDistance{node, node.NodeID.Xor(id).ToInt()})
	}
	Closest1 := unactivelist[0]
	complete := false
	go func() {
		sendrequestchan <- true
		for len(shortlist)<20 && complete == false {
				send := <- readlistchan
				fmt.Println("prepare to send request to nodes"+strconv.Itoa(len(send)))
				for n := 0; n < 3; n++ {
					sendnum := n
					go func(){ 
						fmt.Println("send request to node")
						recived,sender,_ := k.FindNodeQuery(&(send[sendnum].contact),id)
						fmt.Println("recieve from the sender")
						unactivelistchan <- recived
						shortlistchan <- sender
					}()
				}
				timeout := make(chan bool, 1)
				go func() {
					time.Sleep(3e8)
					timeout <- true
				}()
				for  recievenum := 0; recievenum < 3; recievenum++ {
					select{
					case _ = <- countchan:
						fmt.Println("count recieve")
					case _ = <- timeout:
						fmt.Println("timeout")
						break
					}
				}
				sendrequestchan <- true
				fmt.Println("send second request")
		}
		fmt.Println("meet complete?"+ strconv.FormatBool(complete))
		fmt.Println("shortlist filled?"+ strconv.Itoa(len(shortlist)))
		if len(shortlist)<20 && complete == true {
			for i := len(shortlist); i < 20 ;i=i+3 {
				//sendrequestchan <- true
				//fmt.Println("final send request")
				send := <- readlistchan
				for n := 0; n < int(math.Min(float64(3),float64(20-i))); n++ {
					sendnum := n
					go func(){ 
						recived,sender,_ := k.FindNodeQuery(&(send[sendnum].contact),id)
						//fmt.Println("recieve from the sender")
						unactivelistchan <- recived
						shortlistchan <- sender
					}()
				}
				timeout2 := make(chan bool, 1)
				go func() {
					time.Sleep(3e8)
					timeout2 <- true
				}()
				for  recievenum := 0; recievenum < 3; recievenum++ {
					select{
					case _ = <- countchan:
						fmt.Println("count recieve")
					case _ = <- timeout2:
						fmt.Println("timeout")
						break
					}
				}
				sendrequestchan <- true
				fmt.Println("final send request")
			}
		} 
	}()
	go func() {
		fmt.Println("enter go2")
		for {
			select {
			case contactinfo := <- unactivelistchan:
				 fmt.Println("recieve new contacts and update unactivelist")
				 for _,c := range contactinfo{
						unactivelist = append(unactivelist, ContactDistance{c, c.NodeID.Xor(id).ToInt()})
				 }		
				 sort.Sort(ByDist(unactivelist))
				 Closest2 := unactivelist[0]
				 if (Closest2.distance >= Closest1.distance) {
				 	complete = true
				 } else {
				 	Closest1 = Closest2
				 }
				 countchan <- true
			case  request:= <-sendrequestchan:
				if(request == true) {
				    fmt.Println("get request for sending")
				    if len(unactivelist) >= 3 {
				    	sendlist := unactivelist[0:3]
				        unactivelist = unactivelist[3:]
				        fmt.Println("sendlist" + strconv.Itoa(len(sendlist)))
						fmt.Println("unactivelist lenght" + strconv.Itoa(len(unactivelist)))
						readlistchan <- sendlist
						fmt.Println("after sending to readlistchan")
					} 
				}
			case activenode := <-shortlistchan:
				fmt.Println("update shortlist")
				shortlist = append(shortlist,*activenode)
				fmt.Println("after update shortlist" + strconv.Itoa(len(shortlist)))
				if len(shortlist) == 20 {
					resultchan <- shortlist
				}
			}
		}
	}()
	resultContacts := <- resultchan
	return resultContacts,nil
}


func (k *Kademlia) DoIterativeStore(key ID, value []byte) ([]Contact, error) {
	return nil, &CommandFailed{"Not implemented"}
}
func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {
	return nil, &CommandFailed{"Not implemented"}
}

/*
func (k *Kademlia) DoIterativeStore(key ID, value []byte) ([]Contact, error) {
	result := k.DoIterativeFindNode(key)
	for _, c := range result.contacts {
		go k.DoStore(&c, key, value)
	}
	return result,nil
}

func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {
	result := k.DoIterativeFindNode(key)
	if result.value != nil {
		str := "Key: " + result.key.AsString() + " --> Value: " + string(result.value)
		return str
	} else {
		return "Failed"
	}
}*/

// For project 3!
func (k *Kademlia) Vanish(data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	return
}

func (k *Kademlia) Unvanish(searchKey ID) (data []byte) {
	return nil
}

