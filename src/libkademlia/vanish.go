package libkademlia

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	mathrand "math/rand"
	"time"
	"sss"
	"fmt"
	"strconv"
)

type VanashingDataObject struct {
	AccessKey  int64
	Ciphertext []byte
	NumberKeys byte
	Threshold  byte
}

func GenerateRandomCryptoKey() (ret []byte) {
	for i := 0; i < 32; i++ {
		ret = append(ret, uint8(mathrand.Intn(256)))
	}
	return
}

func GenerateRandomAccessKey() (accessKey int64) {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	accessKey = r.Int63()
	return
}

func CalculateSharedKeyLocations(accessKey int64, count int64) (ids []ID) {
	r := mathrand.New(mathrand.NewSource(accessKey))
	ids = make([]ID, count)
	for i := int64(0); i < count; i++ {
		for j := 0; j < IDBytes; j++ {
			ids[i][j] = uint8(r.Intn(256))
		}
	}
	return
}

func encrypt(key []byte, text []byte) (ciphertext []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext = make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], text)
	return
}

func decrypt(key []byte, ciphertext []byte) (text []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext is not long enough")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext
}

func (k *Kademlia) VanishData(vdoID ID, data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	K := GenerateRandomCryptoKey()
	ciphertext := encrypt(K, data)
	vanishmap, err := sss.Split(numberKeys, threshold, ciphertext)
	if err != nil {
		return
	}
	accessKey := GenerateRandomAccessKey()
	ids := CalculateSharedKeyLocations(accessKey, int64(numberKeys))
	i := 0
	for key,value :=  range vanishmap {
		all := append([]byte{key}, value...)
		k.DoIterativeStore(ids[i],all)
		i = i + 1;
	}
	vdo = VanashingDataObject {accessKey, ciphertext, numberKeys, threshold}
	return vdo
}

func (k *Kademlia) UnvanishData(vdo VanashingDataObject) (data []byte) {
	accessKey := vdo.AccessKey
	numberKeys := vdo.NumberKeys
	ids := CalculateSharedKeyLocations(accessKey, int64(numberKeys))
	fmt.Println("!!!!!!!number of ids" + strconv.Itoa(len(ids)))
	unvanishmap := make(map[byte] []byte)
	for _,id := range ids {
		fmt.Println("enter doiterativeValue")
		value, err := k.DoIterativeFindValue(id)
		if err == nil {
			key := value[0]
			v := value[1:]
			unvanishmap[key] = v
	    if len(unvanishmap) == int(vdo.Threshold) {
		 	break
	    }
		}
	}
	K := sss.Combine(unvanishmap)
	fmt.Println("==========recovered key from unvanishmap:")
	fmt.Println(K)
	fmt.Println(len(K))
	data = decrypt(K, vdo.Ciphertext)
	return data
}
