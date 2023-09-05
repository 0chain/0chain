package main
import "fmt"
func main() {
	var myString string
	myString = "abc"
	var myString1 *string
	myString1 = &myString
	*myString1 = "bcd"
	fmt.Println("* myString1: ", *myString1)
	fmt.Println("myString1: ", myString1)
	fmt.Println("&myString1: ", &myString1)
	fmt.Println("&myString: ", &myString)
}