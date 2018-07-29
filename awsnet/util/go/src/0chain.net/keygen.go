package main

import (
	"flag"
	"sort"
	"0chain.net/encryption"
	"fmt"
	"io/ioutil"
	"bytes"
	"encoding/hex"
)

// Type of the client structure
type Client struct {
	client     string
	publicKey  string
	privateKey string
	rank       string
}

// By is the type of a "less" function that defines the ordering of its Planet arguments.
type By func(p1, p2 *Client) bool

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by By) Sort(clients []Client) {
	ps := &clientSorter{
		clients: clients,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

// planetSorter joins a By function and a slice of Planets to be sorted.
type clientSorter struct {
	clients []Client
	by      func(c1, c2 *Client) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (c *clientSorter) Len() int {
	return len(c.clients)
}

// Swap is part of sort.Interface.
func (c *clientSorter) Swap(i, j int) {
	c.clients[i], c.clients[j] = c.clients[j], c.clients[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (c *clientSorter) Less(i, j int) bool {
	return c.by(&c.clients[i], &c.clients[j])
}

func makeMode(legend string, count int, sort bool) []byte {

	// Closures that order the client structure
	publicKeyOrder := func(c1, c2 *Client) bool {
		return c1.publicKey < c2.publicKey
	}

	// TODO: use for future enhancemeents
	_ = publicKeyOrder

	clientOrder := func(c1, c2 *Client) bool {
		return c1.client < c2.client
	}

	var clients = make([]Client, count);
	for i := 0; i < count; i++ {

		publicKey, privateKey, _ := encryption.GenerateKeysBytes()

		client := encryption.Hash(publicKey)

		clients[i].privateKey = hex.EncodeToString(privateKey)
		clients[i].publicKey = hex.EncodeToString(publicKey)
		clients[i].client = client
	}

	if sort == true {
		By(clientOrder).Sort(clients);
	}

	var buffer []byte
	for i := 0; i < count; i++ {
		data := []byte(fmt.Sprintf(" %c%02d: {rank: %02d, client: %v, public: %v, private: %v}\n",
			legend[0],
			i, i, clients[i].client, clients[i].publicKey, clients[i].privateKey))
		buffer = append(buffer, data...)
	}
	return buffer
}

func main() {

	count := flag.Int("count", 3, "Number of Client Ids to generate")
	mode := flag.String("mode", "all", "Mode = miner, sharder, blobber or all")
	sort := flag.Bool("sort", true, "Sort ids by ascending order of clientid")
	keysFile := flag.String("file", "clientid.yml", "Client YAML file to be generated")
	_ = keysFile

	flag.Parse()

	if( *count == 0 )  {
		flag.PrintDefaults()
		return
	}
	// Add the opening section
	var buffer = []byte(fmt.Sprint("---\nclientid:\n"))
	if (*mode == "all") {
		buffer = append(buffer, makeMode("miner", *count, *sort)...)
		buffer = append(buffer, makeMode("sharder", *count, *sort)...)
	} else {
		buffer = append(buffer, makeMode(*mode, *count, *sort)...)
	}

	// Add the end closing section
	buffer = append(buffer, []byte(fmt.Sprint("...\n"))...)
	if len(*keysFile) > 0 {
		ioutil.WriteFile(*keysFile, buffer, 0600 )
	}
	fmt.Printf("%s", bytes.NewBuffer(buffer).String())
}
