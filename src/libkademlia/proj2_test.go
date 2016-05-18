package libkademlia

// import (
// 	//"bytes"
// 	"fmt"
// 	// "math"
// 	//"net"
// 	"strconv"
// 	"testing"
// 	// "time"
// )

// func StringToIpPort(laddr string) (ip net.IP, port uint16, err error) {
// 	hostString, portString, err := net.SplitHostPort(laddr)
// 	if err != nil {
// 		return
// 	}
// 	ipStr, err := net.LookupHost(hostString)
// 	if err != nil {
// 		return
// 	}
// 	for i := 0; i < len(ipStr); i++ {
// 		ip = net.ParseIP(ipStr[i])
// 		if ip.To4() != nil {
// 			break
// 		}
// 	}
// 	portInt, err := strconv.Atoi(portString)
// 	port = uint16(portInt)
// 	return
// }

// EXTRACREDIT
// Check out the correctness of DoIterativeFindNode function
// func TestIterativeFindNode(t *testing.T) {
// 	// tree structure;
// 	// A->B->tree
// 	/*
// 	          C
// 	       /
// 	   A-B -- D
// 	       \
// 	          E
// 	*/
// 	kNum := 40
// 	targetIdx := kNum - 10
// 	instance2 := NewKademlia("localhost:7305")
// 	host2, port2, _ := StringToIpPort("localhost:7305")
// 	//  instance2.DoPing(host2, port2)
// 	tree_node := make([]*Kademlia, kNum)
// 	//t.Log("Before loop")
// 	for i := 0; i < kNum; i++ {
// 		address := "localhost:" + strconv.Itoa(7306+i)
// 		tree_node[i] = NewKademlia(address)
// 		tree_node[i].DoPing(host2, port2)
// 		t.Log("ID:" + tree_node[i].SelfContact.NodeID.AsString())
// 	}
// 	for i := 0; i < kNum; i++ {
// 		if i != targetIdx {
// 			tree_node[targetIdx].DoPing(tree_node[i].SelfContact.Host, tree_node[i].SelfContact.Port)
// 		}
// 	}
// 	SearchKey := tree_node[targetIdx].SelfContact.NodeID
// 	//t.Log("Wait for connect")
// 	//Connect(t, tree_node, kNum)
// 	//t.Log("Connect!")
// 	// time.Sleep(100 * time.Millisecond)
// 	//cHeap := PriorityQueue{instance2.SelfContact, []Contact{}, SearchKey}
// 	//t.Log("Wait for iterative")
// 	res, err := instance2.DoIterativeFindNode(SearchKey)
// 	// res := nil
// 	if err != nil {
// 		t.Error(err.Error())
// 	}
// 	t.Log("SearchKey:" + SearchKey.AsString())
// 	if res == nil || len(res) == 0 {
// 		t.Error("No contacts were found")
// 	}
// 	find := false
// 	fmt.Print("# of results: ")
// 	fmt.Println(len(res))
// 	for _, value := range res {
// 		t.Log(value.NodeID.AsString())
// 		if value.NodeID.Equals(SearchKey) {
// 			find = true
// 		}
// 		//      heap.Push(&cHeap, value)
// 	}
// 	//  c := cHeap.Pop().(Contact)
// 	//  t.Log("Closet Node:" + c.NodeID.AsString())
// 	//  t.Log(strconv.Itoa(cHeap.Len()))
// 	if !find {
// 		t.Log("Instance2:" + instance2.NodeID.AsString())
// 		t.Error("Find wrong id")
// 	}
// 	//t.Error(len(res))
// 	//return
// }

// EXTRACREDIT
//Check out the Correctness of DoIterativeStore
// func TestIterativeStore(t *testing.T) {
// 	// tree structure;
// 	// A->B->tree->tree2
// 	/*
// 	          C
// 	      /
// 	   A-B -- D
// 	       \
// 	          E
// 	*/
// 	instance1 := NewKademlia("localhost:7506")
// 	instance2 := NewKademlia("localhost:7507")
// 	host2, port2, _ := StringToIpPort("localhost:7507")
// 	instance1.DoPing(host2, port2)

// 	//Build the  A->B->Tree structure
// 	tree_node := make([]*Kademlia, 20)
// 	for i := 0; i < 20; i++ {
// 		address := "localhost:" + strconv.Itoa(7508+i)
// 		tree_node[i] = NewKademlia(address)
// 		host_number, port_number, _ := StringToIpPort(address)
// 		instance2.DoPing(host_number, port_number)
// 	}
// 	//implement DoIterativeStore, and get the the result
// 	value := []byte("Hello world")
// 	key := NewRandomID()
// 	contacts, err := instance1.DoIterativeStore(key, value)
// 	//the number of contacts store the value should be 20
// 	if err != nil || len(contacts) != 20 {
// 		t.Error("Error doing DoIterativeStore")
// 	}
// 	//Check all the 22 nodes,
// 	//find out the number of nodes that contains the value
// 	count := 0
// 	// check tree_nodes[0~19]
// 	for i := 0; i < 20; i++ {
// 		result, err := tree_node[i].LocalFindValue(key)
// 		if result != nil && err == nil {
// 			count++
// 		}
// 	}
// 	//check instance2
// 	result, err := instance2.LocalFindValue(key)
// 	if result != nil && err == nil {
// 		count++
// 	}
// 	//check instance1
// 	result, err = instance1.LocalFindValue(key)
// 	if result != nil && err == nil {
// 		count++
// 	}
// 	//Within all 22 nodes
// 	//the number of nodes that store the value should be 20
// 	if count != 20 {
// 		t.Error("DoIterativeStore Failed")
// 	}
// }

// // EXTRACREDIT
//Check out the Correctness of DoIterativeFindValue
// func TestIterativeFindValue(t *testing.T) {
// 	// tree structure;
// 	// A->B->tree->tree2
// 	/*
// 		                F
// 			  /
// 		          C --G
// 		         /    \
// 		       /        H
// 		   A-B -- D
// 		       \
// 		          E
// 	*/

// 	instance1 := NewKademlia("localhost:7406")
// 	instance2 := NewKademlia("localhost:7407")
// 	host2, port2, _ := StringToIpPort("localhost:7407")
// 	instance1.DoPing(host2, port2)

// 	//Build the  A->B->Tree structure
// 	tree_node := make([]*Kademlia, 20)
// 	for i := 0; i < 20; i++ {
// 		address := "localhost:" + strconv.Itoa(7408+i)
// 		tree_node[i] = NewKademlia(address)
// 		host_number, port_number, _ := StringToIpPort(address)
// 		instance2.DoPing(host_number, port_number)
// 	}
// 	//Build the A->B->Tree->Tree2 structure
// 	tree_node2 := make([]*Kademlia, 20)
// 	for j := 20; j < 40; j++ {
// 		address := "localhost:" + strconv.Itoa(7408+j)
// 		tree_node2[j-20] = NewKademlia(address)
// 		host_number, port_number, _ := StringToIpPort(address)
// 		for i := 0; i < 20; i++ {
// 			tree_node[i].DoPing(host_number, port_number)
// 		}
// 	}

// 	//Store value into nodes
// 	value := []byte("Hello world")
// 	key := NewRandomID()
// 	contacts, err := instance1.DoIterativeStore(key, value)
// 	if err != nil || len(contacts) != 20 {
// 		t.Error("Error doing DoIterativeStore")
// 	}

// 	//After Store, check out the correctness of DoIterativeFindValue
// 	result, err := instance1.DoIterativeFindValue(key)
// 	if err != nil || result == nil {
// 		t.Error("Error doing DoIterativeFindValue")
// 	}

// 	//Check the correctness of the value we find
// 	res := string(result[:])
// 	fmt.Println(res)
// 	//t.Error("Finish")
// }



// func TestIterativeFindNode(t *testing.T) {
// 	//r := rand.New(rand.NewSource(time.Now().Unix()))
// 	instanceList := make([]*Kademlia, 0)

// 	for i := 0; i < 200; i++ {
// 		instanceList = append(instanceList, NewKademlia("127.0.0.1:"+strconv.Itoa(8000+i)))
// 	}

// 	counter := 0
// 	for i := 0; i < len(instanceList); i++ {
// 		for j := 0; j < i; j++ {
// 			if i == j || math.Abs(float64(i-j)) > 10 {
// 				continue
// 			}
// 			if counter == 2 {
// 				counter = 0
// 				time.Sleep(10 * time.Millisecond)
// 			}
// 			tmp_host, tmp_port, _ := StringToIpPort("127.0.0.1:" + strconv.Itoa(8000+j))
// 			go instanceList[i].DoPing(tmp_host, tmp_port)
// 			counter++
// 		}
// 	}

// 	target0 := NewRandomID()
// 	target1 := instanceList[100].NodeID
// 	//tmp_host, tmp_port, _ := StringToIpPort("127.0.0.1:" + strconv.Itoa(8000+3))
// 	//instanceList[0].DoPing(tmp_host, tmp_port)

// 	result0 := instanceList[0].IterativeFindNode(target0, false)
// 	result1 := instanceList[0].IterativeFindNode(target1, false)

// 	found0 := false
// 	for _, value := range result0.contacts {
// 		if value.NodeID == target0 {
// 			found0 = true
// 		}
// 	}
// 	found1 := false
// 	for _, value := range result1.contacts {
// 		if value.NodeID == target1 {
// 			found1 = true
// 		}
// 	}
// 	if found0 == false {
// 		// TODO:  calculate all the distance between instancelist to the target
// 		// and sort to see the result is same as the sorted
// 		t.Log("cannot find random target")
// 	}
// 	if found1 == false {
// 		t.Error("Cannot find target")
// 	}

// 	/*
// 		for i := 0; i < len(instanceList); i++ {
// 			for j := 0; j < len(instanceList); j++ {
// 				if i == j {
// 					continue
// 				}
// 				tmp_contact, err := instanceList[i].FindContact(instanceList[j].NodeID)
// 				if err != nil {
// 					t.Errorf("Contact[%d] cannot find contact[%d]\n", i, j)
// 					return
// 				}
// 				if tmp_contact.NodeID != instanceList[j].NodeID {
// 					t.Errorf("Found contact[%d] ID from contact[%d] is not correct\n", j, i)
// 					return
// 				}
// 			}
// 		}
// 	*/
// }