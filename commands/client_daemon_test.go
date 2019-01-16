package commands

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/filecoin-project/go-filecoin/fixtures"
	th "github.com/filecoin-project/go-filecoin/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestListAsks(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	minerDaemon := th.NewDaemon(t,
		th.WithMiner(fixtures.TestMiners[0]),
		th.KeyFile(fixtures.KeyFilePaths()[0]),
		th.DefaultAddress(fixtures.TestAddresses[0]),
	).Start()
	defer minerDaemon.ShutdownSuccess()

	minerDaemon.CreateAsk(minerDaemon, fixtures.TestMiners[0], fixtures.TestAddresses[0], "20", "10")

	listAsksOutput := minerDaemon.RunSuccess("client", "list-asks").ReadStdoutTrimNewlines()
	assert.Equal(fixtures.TestMiners[0]+" 000 20 11", listAsksOutput)
}

func TestStorageDealsAfterRestart(t *testing.T) {
	t.Skip("Temporarily skipped to be fixed in subsequent refactor work")

	minerDaemon := th.NewDaemon(t,
		th.WithMiner(fixtures.TestMiners[0]),
		th.KeyFile(fixtures.KeyFilePaths()[0]),
		th.DefaultAddress(fixtures.TestAddresses[0]),
		th.AutoSealInterval("1"),
	).Start()
	defer minerDaemon.ShutdownSuccess()

	clientDaemon := th.NewDaemon(t,
		th.KeyFile(fixtures.KeyFilePaths()[1]),
		th.DefaultAddress(fixtures.TestAddresses[1]),
	).Start()
	defer clientDaemon.ShutdownSuccess()

	minerDaemon.UpdatePeerID()
	minerDaemon.RunSuccess("mining", "start")

	minerDaemon.ConnectSuccess(clientDaemon)

	minerDaemon.CreateAsk(minerDaemon, fixtures.TestMiners[0], fixtures.TestAddresses[0], "20", "10")
	dataCid := clientDaemon.RunWithStdin(strings.NewReader("HODLHODLHODL"), "client", "import").ReadStdoutTrimNewlines()

	proposeDealOutput := clientDaemon.RunSuccess("client", "propose-storage-deal", fixtures.TestMiners[0], dataCid, "0", "5").ReadStdoutTrimNewlines()

	splitOnSpace := strings.Split(proposeDealOutput, " ")

	dealCid := splitOnSpace[len(splitOnSpace)-1]

	minerDaemon.Restart()
	minerDaemon.RunSuccess("mining", "start")

	clientDaemon.Restart()

	minerDaemon.ConnectSuccess(clientDaemon)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for {
			queryDealOutput := clientDaemon.RunSuccess("client", "query-storage-deal", dealCid).ReadStdout()
			if strings.Contains(queryDealOutput, "posted") {
				wg.Done()
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	th.WaitTimeout(&wg, 120*time.Second)
}
