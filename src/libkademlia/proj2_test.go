package libkademlia

import (
	"bytes"
	"net"
	"strconv"
	"testing"
	"fmt"
	"sort"
	// "time"
)

func StringToIpPort(laddr string) (ip net.IP, port uint16, err error) {
	hostString, portString, err := net.SplitHostPort(laddr)
	if err != nil {
		return
	}
	ipStr, err := net.LookupHost(hostString)
	if err != nil {
		return
	}
	for i := 0; i < len(ipStr); i++ {
		ip = net.ParseIP(ipStr[i])
		if ip.To4() != nil {
			break
		}
	}
	portInt, err := strconv.Atoi(portString)
	port = uint16(portInt)
	return
}

func TestIterativeFindNode(t *testing.T) {
	instance := make([]*Kademlia,30)
  host := make([]net.IP, 30)
  port := make([]uint16, 30)
	for i := 30; i < 60; i++ {
		hostnumber := "localhost:"+strconv.Itoa(7700+i)
		instance[i-30] = NewKademlia(hostnumber)
		host[i-30], port[i-30], _ = StringToIpPort(hostnumber)
	}
	for k := 0; k < 29; k++ {
		instance[k].DoPing(host[k+1], port[k+1])
	}
	instance[29].DoPing(host[0], port[0])
	contact, err := instance[0].DoIterativeFindNode(instance[10].NodeID)
	if err != nil {
		t.Error("node not found ")
		return
	}
	if len(contact) < 20 {
		t.Error("didn't find enough node")
	}
	fmt.Println("Test contact length")
	fmt.Println(len(contact))
	var isFound = false
	for i := 0; i < 20; i++ {
		fmt.Println("Test contact list:" + contact[i].NodeID.AsString())
		if contact[i].NodeID == instance[10].NodeID {
			isFound = true
			return
		}
	}
	fmt.Println("Test target contact:" + instance[10].NodeID.AsString())
	if isFound == false {
		t.Error("cannot find the correct node")
		}
}

func TestIterativeFindNode1(t *testing.T) {
	instance := make([]*Kademlia,30)
    host := make([]net.IP, 30)
    port := make([]uint16, 30)
	for i := 30; i < 60; i++ {
		hostnumber := "localhost:80"+strconv.Itoa(i)
		instance[i-30] = NewKademlia(hostnumber)
		host[i-30], port[i-30], _ = StringToIpPort(hostnumber)
	}
	for k := 0; k < 29; k++ {
		instance[k].DoPing(host[k+1], port[k+1])
	}
	for k := 0; k < 25; k++ {
		instance[k].DoPing(host[k+3], port[k+3])
	}
	for k := 0; k < 20; k++ {
	 	instance[k].DoPing(host[k+5], port[k+5])
	}
	// fmt.Println("After Ping")
	contact, err := instance[0].DoIterativeFindNode(instance[11].NodeID)
	// fmt.Println("After Find")
	// fmt.Println("result lenght:"+strconv.Itoa(len(contact)))
	if err != nil {
		t.Error("node not found ")
		return
	}
	if len(contact) != 20 {
		t.Error("didn't find enough node")
	}
	for i := 0; i < 20; i++ {
	 	if contact[i].NodeID == instance[11].NodeID {
	 		return
	 	}
	 }
	t.Error("cannot find the correct node")
}

func TestIterativeFindValue(t *testing.T) {
	instance := make([]*Kademlia,30)
    host := make([]net.IP, 30)
    port := make([]uint16, 30)
	for i := 30; i < 60; i++ {
		hostnumber := "localhost:81"+strconv.Itoa(i)
		instance[i-30] = NewKademlia(hostnumber)
		host[i-30], port[i-30], _ = StringToIpPort(hostnumber)
	}
	for k := 0; k < 29; k++ {
		instance[k].DoPing(host[k+1], port[k+1])
	}
	for k := 0; k < 25; k++ {
		instance[k].DoPing(host[k+3], port[k+3])
	}
	for k := 0; k < 20; k++ {
		instance[k].DoPing(host[k+5], port[k+5])
	}
	key := instance[11].NodeID
	value := []byte("Hello world")
	err := instance[10].DoStore(&(instance[11].SelfContact), key, value)
    if err != nil {
		t.Error("Could not store value")
	}

	findvalue, err := instance[0].DoIterativeFindValue(key)
	// t.Error(string(findvalue) + "-------------findvalue------\n")
	if err != nil {
		t.Error("value not found ")
		return
	}
	if !bytes.Equal(findvalue, value) {
		t.Error("Stored value did not match found value")
	}
	return
}

func TestIterativeStore(t *testing.T) {
	instance := make([]*Kademlia,30)
	host := make([]net.IP, 30)
	port := make([]uint16, 30)
	for i := 30; i < 60; i++ {
		hostnumber := "localhost:"+strconv.Itoa(7800+i)
		instance[i-30] = NewKademlia(hostnumber)
		host[i-30], port[i-30], _ = StringToIpPort(hostnumber)
	}
	for k := 0; k < 29; k++ {
		instance[k].DoPing(host[k+1], port[k+1])
	}

	key := NewRandomID()
	// value := []byte("Hello world")
	_, err := instance[0].DoIterativeFindNode(key)

	if err != nil {
		t.Error("Do not iterative find correct node to store")
		return
	}

	//find cloest 20 contact
	closeContact := make([]ContactDistance, 0)
	for k := 0; k < 29; k++ {
		dist := key.Xor(instance[k].NodeID).ToInt()
		closeContact = append(closeContact, ContactDistance{instance[k].SelfContact, dist})
	}
	sort.Sort(ByDist(closeContact))

	iterStoreContacts := closeContact[:19]

	for i := 0; i < 30; i++ {
		_, err := instance[i].LocalFindValue(key)

		if err != nil {
			for j := 0; j < 20; j++ {
				if iterStoreContacts[j].contact.NodeID == instance[i].SelfContact.NodeID {    //can add value check
					break
				} else {
					t.Error("Do not store value in correct node")
					return
				}
			}
		}
	}
}

func TestUnvanishSimple(t *testing.T) {
	instance1 := NewKademlia("localhost:7890")
	instance2 := NewKademlia("localhost:7891")
	instance3 := NewKademlia("localhost:7892")
	host2, port2, _ := StringToIpPort("localhost:7891")
	host3, port3, _ := StringToIpPort("localhost:7892")
	instance1.DoPing(host2, port2)
	instance1.DoPing(host3, port3)
	instance3.DoPing(host2, port2)

	vdoID := NewRandomID()
	data := []byte("Hello world")
	numberKeys := 3
	threshold := 2
	vdo := instance1.Vanish(vdoID, data, byte(numberKeys), byte(threshold), 300)
	if vdo.Ciphertext == nil {
		t.Error("Could not vanish vdo")
	}
	// fmt.Println("==========instance[0].SelfContact.Host:")
	// fmt.Println(instance[0].SelfContact.Host)
	//contact, err := instance[10].DoIterativeFindNode(instance[0].NodeID)
    dataUnvanished := instance2.Unvanish(instance1.NodeID, vdoID)
	if dataUnvanished == nil {
		t.Error("Could not vanish vdo")
	}
	// if !bytes.Equal(newData, data) {
	// 	t.Error("Unvanish wrong data")
 // 	}

	return
}
