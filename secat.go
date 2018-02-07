package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
)

// the number of bits for our DHKE primes
const bits = 128

func main() {
	// remember this is a pointer
	var serv = flag.Bool("l", false, "enable server mode (listen)")
	var verbose = flag.Bool("v", false, "verbose mode")
	var udp = flag.Bool("u", false, "use UDP instead of TCP")
	flag.Parse()
	args := flag.Args()
	/* end argument parsing */
	if *serv {
		server(args, *udp, *verbose)
	} else {
		client(args, *udp, *verbose)
	}
}

func client(args []string, udp, vb bool) {
	if len(args) < 2 {
		fmt.Printf("Error: No ports specified for connection\n")
		os.Exit(1)
	}
	// force IPv4 for now
	var proto string
	if udp {
		proto = "udp4"
	} else {
		proto = "tcp4"
	}
	addr := fmt.Sprintf("%s:%s", args[0], args[1])
	conn, err := net.Dial(proto, addr)
	handle(err)
	defer conn.Close()
	if vb {
		fmt.Println("starting client")
	}
	base(conn)
}

func server(args []string, udp, vb bool) {
	// maybe support random port generation someday
	if len(args) < 1 {
		fmt.Printf("Error: No listen port specified\n")
		os.Exit(1)
	}
	// going to stick with IPv4 for now
	var proto string
	addr := fmt.Sprintf("localhost:%s", args[0])
	var conn net.Conn
	if udp {
		proto = "udp4"
		uaddr, err := net.ResolveUDPAddr(proto, addr)
		handle(err)
		conn, err = net.ListenUDP(proto, uaddr)
		handle(err)
	} else {
		proto = "tcp4"
		listener, err := net.Listen(proto, addr)
		handle(err)
		defer listener.Close()
		conn, err = listener.Accept()
		handle(err)
	}
	defer conn.Close()
	if vb {
		fmt.Println("starting server")
	}
	base(conn)
}

func base(conn net.Conn) {
	connbuf := bufio.NewReader(conn)
	connwr := bufio.NewWriter(conn)
	psRead := bufio.NewReader(os.Stdin)
	psWrite := bufio.NewWriter(os.Stdout)
	go func() {
		for {
			txt, err := psRead.ReadString('\n')
			handle(err)
			// can also use Write to write []byte
			connwr.WriteString(txt)
			connwr.Flush()
		}
	}()

	kill := make(chan bool)

	go func() {
		for {
			// can also use ReadSlice to get a []byte
			str, err := connbuf.ReadString('\n')
			if len(str) > 0 {
				psWrite.WriteString(str)
				psWrite.Flush()
			}
			handle(err)
		}
	}()
	// ghetto block for connection to end
	<-kill
}

func cryptoSetup() {
	// define our prime for dhke
	myPrime, err := rand.Prime(rand.Reader, bits)
	handle(err)
	// get the public data to send to server
	ourPrime, ourMod := genDH()
	// compute the secret key for use with aes
	secKey := big.NewInt(0)
	secKey.Exp(ourPrime, myPrime, ourMod)
	// we will want to call secKey.Bytes for use as AES key later
	// debug
	fmt.Printf("debug: %v %v %v %v\n", myPrime, ourPrime, ourMod, secKey.BitLen())

}

// CTRMode implements the counter mode for AES encryption/decryption
//   on the given data streams
//   inspired from https://golang.org/src/crypto/cipher/example_test.go
//
func CTRMode() {
	key := []byte("example key 1234")
	plaintext := []byte("some plaintext")

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	fmt.Printf("%v\n", hex.EncodeToString(ciphertext[aes.BlockSize:]))

	// It's important to remember that ciphertexts must be authenticated
	// (i.e. by using crypto/hmac) as well as being encrypted in order to
	// be secure.

	// CTR mode is the same for both encryption and decryption, so we can
	// also decrypt that ciphertext with NewCTR.

	plaintext2 := make([]byte, len(plaintext))
	stream = cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext2, ciphertext[aes.BlockSize:])

	fmt.Printf("%s\n", plaintext2)
	// Output: some plaintext
}

/* genDH generates the public base and modulus for DHKE */
func genDH() (*big.Int, *big.Int) {
	// define the public base prime
	ourPrime, err := rand.Prime(rand.Reader, bits)
	handle(err)
	// define the public modulus
	ourMod, err := rand.Prime(rand.Reader, bits)
	handle(err)
	return ourPrime, ourMod
}

/* generic error handler */
func handle(e error) {
	if e != nil {
		log.Fatal(e)
	}
}