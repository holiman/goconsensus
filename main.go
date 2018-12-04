package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	common2 "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/mobile"
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
	},
	"Homestead": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      2000,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
	},
	"EIP150": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
	},
	"EIP158": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
	},
	"Byzantium": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      0,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
	},
	"Constantinople": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      0,
		"HIVE_FORK_CONSTANTINOPLE": 0,
	},
	"FrontierToHomesteadAt5": {
		"HIVE_FORK_HOMESTEAD":      5,
		"HIVE_FORK_TANGERINE":      2000,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
	},
	"HomesteadToEIP150At5": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      5,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
	},
	"HomesteadToDaoAt5": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      2000,
		"HIVE_FORK_SPURIOUS":       2000,
		"HIVE_FORK_DAO_BLOCK":      5,
		"HIVE_FORK_BYZANTIUM":      2000,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
	},
	"EIP158ToByzantiumAt5": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      5,
		"HIVE_FORK_CONSTANTINOPLE": 2000,
	},
	"ByzantiumToConstantinopleAt5": {
		"HIVE_FORK_HOMESTEAD":      0,
		"HIVE_FORK_TANGERINE":      0,
		"HIVE_FORK_SPURIOUS":       0,
		"HIVE_FORK_DAO_BLOCK":      2000,
		"HIVE_FORK_BYZANTIUM":      0,
		"HIVE_FORK_CONSTANTINOPLE": 5,
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
				t := &Testcase{blockTest: &blocktest, name: name}
				if err := t.validate(); err != nil {
					log.Error("error", "err", err, "test", t.name)
					continue
				}
				i = i + 1
				out <- t
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
	err       error
	executed  bool
	name      string
	blockTest *BlockTest
	nodeId    string
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
	if err := os.Mkdir(fmt.Sprintf("./%s", t.name), 0700); err != nil {
		return "", "", "", err
	}
	if err := os.Mkdir(fmt.Sprintf("./%s/blocks", t.name), 0700); err != nil {
		return "", "", "", err
	}
	genesis := toGethGenesis(&(t.blockTest.json))
	genBytes, _ := json.Marshal(genesis)
	genesisFile := fmt.Sprintf("./%v/genesis.json", t.name)
	if err := ioutil.WriteFile(genesisFile, genBytes, 0777); err != nil {
		return "", "", "", fmt.Errorf("failed writing genesis: %v", err)
	}
	blockFolder := fmt.Sprintf("./%s/blocks", t.name)
	for i, block := range t.blockTest.json.Blocks {
		rlpdata := common2.FromHex(block.Rlp)
		fname := fmt.Sprintf("%s/%04d.rlp", blockFolder, i+1)
		if err := ioutil.WriteFile(fname, rlpdata, 0777); err != nil {
			return "", "", "", fmt.Errorf("failed writing block %d: %v", i, err)
		}
	}
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
		return fmt.Errorf("last block mismatch, expectd 0x%x got 0x%x", exp, got)
	}
	return nil
}
func (t *Testcase) details() string {
	if t.err != nil {
		return t.err.Error()
	}
	return ""
}

func (t *Testcase) finished(err error) {
	t.err = err
	t.executed = true
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
	log.Info("starting test", "name", t.name)
	start := time.Now()
	var (
		err error
	)
	var done = func(err error) {
		t.finished(err)
		log.Info("reporting", "id", t.nodeId)
		if len(t.nodeId) > 0 {
			if err = be.api.AddResults(t.err == nil, t.nodeId, t.name, t.details(), time.Since(start)); err != nil {
				log.Info("errors occurred when adding results", "err", err)
			}
			// No longer needed to call KillNode, it gets terminated in AddResults
			//be.api.KillNode(t.nodeId)
		}
		log.Info("reporting done", "id", t.nodeId)
	}
	defer done(err)
	genesis, _, blocks, err := t.artefacts()
	if err != nil {
		return err
	}
	env := map[string]string{
		"CLIENT":             clientType,
		"HIVE_INIT_GENESIS":  genesis,
		"HIVE_INIT_BLOCKS":   blocks,
		"HIVE_FORK_DAO_VOTE": "1",
	}
	t.updateEnv(env)

	// spin up a node
	log.Info("requesting new node")
	nodeid, ip, err := be.api.StartNewNode(env)
	if err != nil {
		return err
	}
	log.Info("node up", "id", nodeid, "ip", ip)
	t.nodeId = nodeid
	client, err := geth.NewEthereumClient(fmt.Sprintf("http://%s:8545", ip.String()))
	if err != nil {
		return err
	} // set version

	//v, err := client.EthereumClient.getVersion()
	//if err != nil {
	//	return err
	//}

	// verify preconditions
	ctx := geth.NewContext().WithTimeout(int64(10 * time.Second))
	log.Info("checking precond", "id", nodeid)
	nodeGenesis, err := client.GetBlockByNumber(ctx, 0)
	log.Info("checking precond ok", "id", nodeid)
	if err != nil {
		err = fmt.Errorf("failed to check genesis: %v", err)
		return err
	}
	gotHash := nodeGenesis.GetHash()
	if gotHash == nil {
		return fmt.Errorf("got nil genesis")
	}

	if err = t.verifyGenesis((*gotHash).GetBytes()); err != nil {
		return err
	}
	// verify postconditions
	log.Info("checking postcond", "id", nodeid)
	ctx = geth.NewContext().WithTimeout(int64(10 * time.Second))
	lastBlock, err := client.GetBlockByNumber(ctx, -1)
	log.Info("checking postcond ok", "id", nodeid)
	if err != nil {
		return err
	}
	if err = t.verifyBestblock(lastBlock.GetHash().GetBytes()); err != nil {
		return err
	}
	return nil
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
	for i := 0; i < runtime.GOMAXPROCS(-1); i++ {
		wg.Add(1)
		go func() {
			b := BlocktestExecutor{api: host, clients: availableClients}
			b.run(testCh)
			wg.Done()
		}()
	}
	log.Info("Tests started", "num threads", runtime.GOMAXPROCS(-1))
	wg.Wait()
}
