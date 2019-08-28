package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	common2 "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/hive/simulators/common"
)

type envvars map[string]int

var ruleset = map[string]envvars{
	"Frontier": {
		"HIVE_FORK_HOMESTEAD":      2000,
		"HIVE_FORK_TANGERINE":      2000,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"Homestead": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      2000,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"EIP150": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"EIP158": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"Byzantium": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      0,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"Constantinople": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      0,
		"HIVE_FORK_CONSTANTINOPLE": 0,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"ConstantinopleFix": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      0,
		"HIVE_FORK_CONSTANTINOPLE": 0,
		"HIVE_FORK_PETERSBURG":     0,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"Istanbul": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      0,
		"HIVE_FORK_CONSTANTINOPLE": 0,
		"HIVE_FORK_PETERSBURG":     0,
		"HIVE_FORK_ISTANBUL":       0,
	},
	"FrontierToHomesteadAt5": {
		"HIVE_FORK_HOMESTEAD":      5,
		"HIVE_FORK_TANGERINE":      2000,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"HomesteadToEIP150At5": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      5,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"HomesteadToDaoAt5": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      2000,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      5,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"EIP158ToByzantiumAt5": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      5,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"ByzantiumToConstantinopleAt5": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      0,
		"HIVE_FORK_CONSTANTINOPLE": 5,
		"HIVE_FORK_PETERSBURG":     2000,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"ByzantiumToConstantinopleFixAt5": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      0,
		"HIVE_FORK_CONSTANTINOPLE": 5,
		"HIVE_FORK_PETERSBURG":     5,
		"HIVE_FORK_ISTANBUL":       2000,
	},
	"ConstantinopleFixToIstanbulAt5": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      0,
		"HIVE_FORK_CONSTANTINOPLE": 0,
		"HIVE_FORK_PETERSBURG":     0,
		"HIVE_FORK_ISTANBUL":       5,
	},
}

func deliverTests(root string) chan *Testcase {
	out := make(chan *Testcase)
	var i, j = 0, 0
	go func() {
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if fname := info.Name(); !strings.HasSuffix(fname, ".json") {
				return nil
			}
			tests := make(map[string]BlockTest)
			data, err := ioutil.ReadFile(path)
			if err = json.Unmarshal(data, &tests); err != nil {
				log.Error("error", "err", err)
				return err
			}
			j = j + 1
			for name, blocktest := range tests {
				// t is declared explicitly here, if implicit := - declaration is used,
				// golang will reuse the underlying object, and overwrite the object while it's being tested
				// by a separate thread.
				// That is also the reason that blocktest within the struct is by-value instead of by-reference
				var t Testcase
				t = Testcase{blockTest: blocktest, name: name, filepath: path}
				if err := t.validate(); err != nil {
					log.Error("error", "err", err, "test", t.name)
					continue
				}
				i = i + 1
				out <- &t
			}
			return nil
		})
		log.Info("file iterator done", "files", j, "tests", i)
		close(out)
	}()
	return out
}

type BlocktestExecutor struct {
	api     common.SimulatorAPI
	clients []string
}

type Testcase struct {
	name      string
	blockTest BlockTest
	nodeId    string
	filepath  string
}

// validate returns error if the network is not defined
func (t *Testcase) validate() error {
	net := t.blockTest.json.Network
	if _, exist := ruleset[net]; !exist {
		return fmt.Errorf("network %v not defined in ruleset", net)
	}
	return nil
}

// updateEnv sets environment variables from the test
func (t *Testcase) updateEnv(env map[string]string) {
	// Environment variables for rules
	rules := ruleset[t.blockTest.json.Network]
	for k, v := range rules {
		env[k] = fmt.Sprintf("%d", v)
	}
	// Possibly disable POW
	if t.blockTest.json.SealEngine == "NoProof" {
		env["HIVE_SKIP_POW"] = "1"
	}
}

func toGethGenesis(test *btJSON) *core.Genesis {
	genesis := &core.Genesis{
		Nonce:      test.Genesis.Nonce.Uint64(),
		Timestamp:  test.Genesis.Timestamp.Uint64(),
		ExtraData:  test.Genesis.ExtraData,
		GasLimit:   test.Genesis.GasLimit,
		Difficulty: test.Genesis.Difficulty,
		Mixhash:    test.Genesis.MixHash,
		Coinbase:   test.Genesis.Coinbase,
		Alloc:      test.Pre,
	}
	return genesis
}

func (t *Testcase) artefacts() (string, string, string, error) {

	key := fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("%s%s", t.filepath, t.name))))
	blockFolder := fmt.Sprintf("./%s/blocks", key)

	if err := os.Mkdir(fmt.Sprintf("./%s", key), 0700); err != nil {
		return "", "", "", err
	}
	if err := os.Mkdir(blockFolder, 0700); err != nil {
		return "", "", "", err
	}
	genesis := toGethGenesis(&(t.blockTest.json))
	genBytes, _ := json.Marshal(genesis)
	genesisFile := fmt.Sprintf("./%v/genesis.json", key)
	if err := ioutil.WriteFile(genesisFile, genBytes, 0777); err != nil {
		return "", "", "", fmt.Errorf("failed writing genesis: %v", err)
	}

	for i, block := range t.blockTest.json.Blocks {
		rlpdata := common2.FromHex(block.Rlp)
		fname := fmt.Sprintf("%s/%04d.rlp", blockFolder, i+1)
		if err := ioutil.WriteFile(fname, rlpdata, 0777); err != nil {
			return "", "", "", fmt.Errorf("failed writing block %d: %v", i, err)
		}
	}
	//log.Info("Test artefacts", "testname", t.name, "testfile", t.filepath, "blockfolder", blockFolder)
	return genesisFile, "", blockFolder, nil
}

func (t *Testcase) verifyGenesis(got []byte) error {
	if exp := t.blockTest.json.Genesis.Hash; bytes.Compare(exp[:], got) != 0 {
		return fmt.Errorf("genesis mismatch, expectd 0x%x got 0x%x", exp, got)
	}
	return nil
}
func (t *Testcase) verifyBestblock(got []byte) error {
	if exp := t.blockTest.json.BestBlock; bytes.Compare(exp[:], got) != 0 {
		return fmt.Errorf("last block mismatch, expectd 0x%x got 0x%x (%v %v)", exp, got, t.name, t.filepath)
	}
	return nil
}

func (be *BlocktestExecutor) run(testChan chan *Testcase) {
	var i = 0
	for t := range testChan {
		for _, client := range be.clients {
			if err := be.runTest(t, client); err != nil {
				log.Error("error", "err", err)
			}
			i += 1
		}
	}
	log.Info("executor finished", "num_executed", i)
}

func (be *BlocktestExecutor) runTest(t *Testcase, clientType string) error {
	// get the artefacts
	log.Info("Starting test", "name", t.name, "file", t.filepath)
	start := time.Now()
	var (
		err error
	)
	var done = func() {
		var (
			errString = ""
			success   = (err == nil)
		)
		if !success {
			errString = err.Error()
		}
		if id := t.nodeId; id != "" {
			log.Info("reporting", "id", t.nodeId, "err", err)
			testname := fmt.Sprintf("%s:%s", t.filepath, t.name)
			if strings.HasPrefix(testname, "/tests/") {
				testname = fmt.Sprintf(".%s", testname)
			}
			if err = be.api.AddResults(success, id, testname, errString, time.Since(start)); err != nil {
				log.Info("errors occurred when adding results", "err", err)
			}
		} else {
			log.Info("Error occurred, but no node to report to", "test", t.name, "err", err)
		}
	}
	defer done()
	genesis, _, blocks, err := t.artefacts()
	t1 := time.Now()
	if err != nil {
		return err
	}
	env := map[string]string{
		"CLIENT":             clientType,
		"HIVE_INIT_GENESIS":  genesis,
		"HIVE_INIT_BLOCKS":   blocks,
		"HIVE_FORK_DAO_VOTE": "1",
		// If we don't supply these, hive will spin up a temporary container to copy
		// default-values from
		"HIVE_INIT_CHAIN": "ignore",
		"HIVE_INIT_KEYS":  "ignore",
	}
	t.updateEnv(env)
	// spin up a node
	nodeid, ip, _, err := be.api.StartNewNode(env)
	t2 := time.Now()
	if err != nil {
		return err
	}
	t.nodeId = nodeid
	ctx := context.Background()
	rawClient, err := rpc.DialContext(ctx, fmt.Sprintf("http://%s:8545", ip.String()))
	if err != nil {
		err = fmt.Errorf("failed to start client: %v", err)
		return err
	}
	genesisHash, err := getHash(rawClient, hexutil.EncodeBig(new(big.Int)))
	if err != nil {
		err = fmt.Errorf("failed to check genesis: %v", err)
		return err
	}
	t3 := time.Now()

	if err = t.verifyGenesis(genesisHash); err != nil {
		return err
	}
	t4 := time.Now()
	// verify postconditions
	lastHash, err := getHash(rawClient, "latest")
	if err != nil {
		return err
	}
	t5 := time.Now()
	if err = t.verifyBestblock(lastHash); err != nil {
		return err
	}
	t6 := time.Now()
	log.Info("Test done", "name", t.name, "artefacts", t1.Sub(start), "newnode", t2.Sub(t1), "getGenesis", t3.Sub(t2), "verifyGenesis", t4.Sub(t3), "getLatest", t5.Sub(t4), "verifyLatest", t6.Sub(t5))
	return nil
}

func getHash(rawClient *rpc.Client, arg string) ([]byte, error) {
	blockData := make(map[string]interface{})
	if err := rawClient.Call(&blockData, "eth_getBlockByNumber", arg, false); err != nil {
		// Make one more attempt
		log.Info("Client connect failed, making one more attempt...")
		if err = rawClient.Call(&blockData, "eth_getBlockByNumber", arg, false); err != nil {
			return nil, err
		}
	}
	if hash, exist := blockData["hash"]; !exist {
		return nil, fmt.Errorf("no hash found in response")
	} else {
		if hexHash, ok := hash.(string); !ok {
			return nil, fmt.Errorf("error: string conversion failed for %v", hash)
		} else {
			return common2.HexToHash(hexHash).Bytes(), nil
		}
	}
}
func main() {
	hivesim, isset := os.LookupEnv("HIVE_SIMULATOR")

	if !isset {
		log.Error("simulator API not set ($HIVE_SIMULATOR)")
		os.Exit(1)
	}
	log.Info("Hive simulator", "url", hivesim)

	testpath, isset := os.LookupEnv("TESTPATH")
	if !isset {
		log.Error("Test path not set ($TESTPATH)")
		os.Exit(1)
	}

	//Try to connect to the simulator host and get the client list
	host := &common.SimulatorHost{
		HostURI: &hivesim,
	}
	availableClients, _ := host.GetClientTypes()
	log.Info("Got clients", "clients", availableClients)
	fileRoot := fmt.Sprintf("%s/BlockchainTests/", testpath)
	testCh := deliverTests(fileRoot)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			b := BlocktestExecutor{api: host, clients: availableClients}
			b.run(testCh)
			wg.Done()
		}()
	}
	log.Info("Tests started", "num threads", 16)
	wg.Wait()
}
