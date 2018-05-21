package config

/*Config - all the config options passed from the command line*/
type Config struct {
	Host     string
	Port     int
	ChainID  string
	TestMode bool
}

/*Configuration of the system */
var Configuration Config

/*TestNet is the program running in TestNet mode? */
func TestNet() bool {
	return Configuration.TestMode
}
