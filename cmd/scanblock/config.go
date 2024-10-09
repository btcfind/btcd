package main

import "github.com/jessevdk/go-flags"

type Config struct {
	RPCUser   string `long:"rpcuser" description:"RPC username"`
	RPCPass   string `long:"rpcpass" description:"RPC password"`
	RPCListen string `long:"rpclisten" description:"RPC server address"`
	RPCCert   string `long:"rpccert" description:"RPC server certificate file"`
	DBPath    string `long:"dbpath" description:"Path to LevelDB database" default:"./data/ldb"`
}

type Options struct {
	ConfigFile string `short:"c" long:"config" description:"Path to configuration file" default:"scan.conf"`
}

func loadConfig(filename string) (*Config, error) {
	config := &Config{}
	parser := flags.NewParser(config, flags.Default)
	err := flags.NewIniParser(parser).ParseFile(filename)
	if err != nil {
		return nil, err
	}
	return config, nil
}
