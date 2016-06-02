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

func TestPing(t *testing.T) {
	instance1 := NewKademlia("localhost:7890")
	instance2 := NewKademlia("localhost:7891")
	host2, port2, _ := StringToIpPort("localhost:7891")
	contact2, err := instance2.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("A node cannot find itself's contact info")
	}
	contact2, err = instance2.FindContact(instance1.NodeID)
	if err == nil {
		t.Error("Instance 2 should not be able to find instance " +
			"1 in its buckets before ping instance 1")
	}
	instance1.DoPing(host2, port2)
	contact2, err = instance2.FindContact(instance1.NodeID)
	if err != nil {
		t.Error("Instance 2 should be able to find instance" +
			"1 in its buckets before ping instance 1")
	}
	contact2, err = instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	wrong_ID := NewRandomID()
	_, err = instance2.FindContact(wrong_ID)
	if err == nil {
		t.Error("Instance 2 should not be able to find a node with the wrong ID")
	}

	contact1, err := instance2.FindContact(instance1.NodeID)
	if err != nil {
		t.Error("Instance 1's contact not found in Instance 2's contact list")
		return
	}
	if contact1.NodeID != instance1.NodeID {
		t.Error("Instance 1 ID incorrectly stored in Instance 2's contact list")
	}
	if contact2.NodeID != instance2.NodeID {
		t.Error("Instance 2 ID incorrectly stored in Instance 1's contact list")
	}
	return
}

func TestStore(t *testing.T) {
	// test Dostore() function and LocalFindValue() function
	instance1 := NewKademlia("localhost:7892")
	instance2 := NewKademlia("localhost:7893")
	host2, port2, _ := StringToIpPort("localhost:7893")
	instance1.DoPing(host2, port2)
	contact2, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	key := NewRandomID()
	value := []byte("Hello World")
	err = instance1.DoStore(contact2, key, value)
	if err != nil {
		t.Error("Can not store this value")
	}
	storedValue, err := instance2.LocalFindValue(key)
	if err != nil {
		t.Error("Stored value not found!")
	}
	if !bytes.Equal(storedValue, value) {
		t.Error("Stored value did not match found value")
	}
	return
}

func TestFindNode(t *testing.T) {
	// tree structure;
	// A->B->tree
	/*
	         C
	      /
	  A-B -- D
	      \
	         E
	*/
	instance1 := NewKademlia("localhost:7894")
	instance2 := NewKademlia("localhost:7895")
	host2, port2, _ := StringToIpPort("localhost:7895")
	instance1.DoPing(host2, port2)
	contact2, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}
	tree_node := make([]*Kademlia, 10)
	for i := 0; i < 10; i++ {
		address := "localhost:" + strconv.Itoa(7896+i)
		tree_node[i] = NewKademlia(address)
		host_number, port_number, _ := StringToIpPort(address)
		instance2.DoPing(host_number, port_number)
	}
	key := NewRandomID()
	contacts, err := instance1.DoFindNode(contact2, key)
	if err != nil {
		t.Error("Error doing FindNode")
	}

	if contacts == nil || len(contacts) == 0 {
		t.Error("No contacts were found")
	}
	// TODO: Check that the correct contacts were stored
	//       (and no other contacts)

	for i := 0; i < 10; i++ {
		contacts, err := instance1.DoFindNode(contact2, tree_node[i].NodeID)
		if err != nil {
			t.Error("Contact not find")
		}
		contactTemp := contacts[0]
		if contactTemp.NodeID.Compare(tree_node[i].NodeID) != 0{
			t.Error("Contacts not correctly stored")
		}
	}

	return
}

func TestFindValue(t *testing.T) {
	// tree structure;
	// A->B->tree
	/*
	         C
	      /
	  A-B -- D
	      \
	         E
	*/
	instance1 := NewKademlia("localhost:7926")
	instance2 := NewKademlia("localhost:7927")
	host2, port2, _ := StringToIpPort("localhost:7927")
	instance1.DoPing(host2, port2)
	contact2, err := instance1.FindContact(instance2.NodeID)
	if err != nil {
		t.Error("Instance 2's contact not found in Instance 1's contact list")
		return
	}

	tree_node := make([]*Kademlia, 10)
	for i := 0; i < 10; i++ {
		address := "localhost:" + strconv.Itoa(7928+i)
		tree_node[i] = NewKademlia(address)
		host_number, port_number, _ := StringToIpPort(address)
		instance2.DoPing(host_number, port_number)
	}

	key := NewRandomID()
	value := []byte("Hello world")
	err = instance2.DoStore(contact2, key, value)
	if err != nil {
		t.Error("Could not store value")
	}

	// Given the right keyID, it should return the value
	foundValue, contacts, err := instance1.DoFindValue(contact2, key)
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	//Given the wrong keyID, it should return k nodes.
	wrongKey := NewRandomID()
	foundValue, contacts, err = instance1.DoFindValue(contact2, wrongKey)
	if contacts == nil || len(contacts) < 10 {
		t.Error("Searching for a wrong ID did not return contacts")
	}

	// TODO: Check that the correct contacts were stored
	//       (and no other contacts)
	for i := 0; i < 10; i++ {
		contacts, err := instance1.DoFindNode(contact2, tree_node[i].NodeID)
		if err != nil {
			t.Error("Contact not find")
		}
		contactTemp := contacts[0]
		if contactTemp.NodeID.Compare(tree_node[i].NodeID) != 0{
			t.Error("Contacts not correctly stored")
		}
	}
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

func TestVanish(t *testing.T) {
	instance := make([]*Kademlia,30)
	host := make([]net.IP, 30)
	port := make([]uint16, 30)
	for i := 30; i < 60; i++ {
		hostnumber := "localhost:"+strconv.Itoa(7200+i)
		instance[i-30] = NewKademlia(hostnumber)
		host[i-30], port[i-30], _ = StringToIpPort(hostnumber)
	}
	for k := 0; k < 29; k++ {
		instance[k].DoPing(host[k+1], port[k+1])
	}

	vdoID := NewRandomID()
	data := []byte("Hello world")
	numberKeys := 4
	threshold := 3
	vdo := instance[0].Vanish(vdoID, data, byte(numberKeys), byte(threshold), 300)
	if vdo.Ciphertext == nil {
		t.Error("Could not vanish vdo")
	}
}

func TestUnvanish(t *testing.T) {
	instance := make([]*Kademlia,30)
	host := make([]net.IP, 30)
	port := make([]uint16, 30)
	for i := 30; i < 60; i++ {
		hostnumber := "localhost:"+strconv.Itoa(7100+i)
		instance[i-30] = NewKademlia(hostnumber)
		host[i-30], port[i-30], _ = StringToIpPort(hostnumber)
	}
	for k := 0; k < 29; k++ {
		instance[k].DoPing(host[k+1], port[k+1])
	}

	vdoID := NewRandomID()
	data := []byte("Hello world")
	numberKeys := 4
	threshold := 3
	vdo := instance[0].Vanish(vdoID, data, byte(numberKeys), byte(threshold), 300)
	if vdo.Ciphertext == nil {
		t.Error("Could not vanish vdo")
	}
	//contact, err := instance[10].DoIterativeFindNode(instance[0].NodeID)
	_ = instance[0].Unvanish(instance[0].NodeID, vdoID)
	return
}