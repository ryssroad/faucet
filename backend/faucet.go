package main

import (
	"encoding/json"
	"fmt"
	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/joho/godotenv"
	"github.com/tendermint/tmlibs/bech32"
	"github.com/tomasen/realip"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var chain string
var recaptchaSecretKey string
var amountFaucet string
var amountSteak string
var key string
var pass string
var node string
var publicUrl string

type claim_struct struct {
	Address  string
	Response string
}

func getEnv(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		fmt.Println(key, "=", value)
		return value
	} else {
		log.Fatal("Error loading environment variable: ", key)
		return ""
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	chain = getEnv("FAUCET_CHAIN")
	recaptchaSecretKey = getEnv("FAUCET_RECAPTCHA_SECRET_KEY")
	amountFaucet = getEnv("FAUCET_AMOUNT_FAUCET")
	amountSteak = getEnv("FAUCET_AMOUNT_STEAK")
	key = getEnv("FAUCET_KEY")
	pass = getEnv("FAUCET_PASS")
	node = getEnv("FAUCET_NODE")
	publicUrl = getEnv("FAUCET_PUBLIC_URL")

	recaptcha.Init(recaptchaSecretKey)

	http.HandleFunc("/claim", getCoinsHandler)

	if err := http.ListenAndServe(publicUrl, nil); err != nil {
		log.Fatal("failed to start server", err)
	}
}

func executeCmd(command string, writes ...string) {
	cmd, wc, _ := goExecute(command)

	for _, write := range writes {
		wc.Write([]byte(write + "\n"))
	}
	cmd.Wait()
}

func goExecute(command string) (cmd *exec.Cmd, pipeIn io.WriteCloser, pipeOut io.ReadCloser) {
	cmd = getCmd(command)
	pipeIn, _ = cmd.StdinPipe()
	pipeOut, _ = cmd.StdoutPipe()
	go cmd.Start()
	time.Sleep(time.Second)
	return cmd, pipeIn, pipeOut
}

func getCmd(command string) *exec.Cmd {
	// split command into command and args
	split := strings.Split(command, " ")

	var cmd *exec.Cmd
	if len(split) == 1 {
		cmd = exec.Command(split[0])
	} else {
		cmd = exec.Command(split[0], split[1:]...)
	}

	return cmd
}

func getCoinsHandler(w http.ResponseWriter, request *http.Request) {
	var claim claim_struct
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fmt.Println("we have received new message")

	// decode JSON response from front end
	decoder := json.NewDecoder(request.Body)
	decoderErr := decoder.Decode(&claim)
	if decoderErr != nil {
		panic(decoderErr)
	}
	fmt.Println("test1")

	// make sure address is bech32
	readableAddress, decodedAddress, decodeErr := bech32.DecodeAndConvert(claim.Address)
	if decodeErr != nil {
		panic(decodeErr)
	}
	// re-encode the address in bech32
	encodedAddress, encodeErr := bech32.ConvertAndEncode(readableAddress, decodedAddress)
	if encodeErr != nil {
		panic(encodeErr)
	}
	fmt.Println("test2")

	// make sure captcha is valid
	clientIP := realip.FromRequest(request)
	captchaResponse := claim.Response
	captchaPassed, captchaErr := recaptcha.Confirm(clientIP, captchaResponse)
	if captchaErr != nil {
		panic(captchaErr)
	}
	fmt.Println("test3")

	// send the coins!
	if captchaPassed {
		// demo erc20 address: ab4c7f7184a2362048576a312d0b0257bc44e070
		// teleport tx bank send validator0 teleport14dx87uvy5gmzqjzhdgcj6zcz277yfcrsmcws6m 100000000000000000000atele --gas-prices 5000000000atele  --node tcp://localhost:26657 --chain-id teleport_7001-1 --keyring-backend test --home ~/teleport_testnet/validators/validator0/teleport -y
		sendFaucet := fmt.Sprintf(
			"teleport tx bank send validator0 %v %v --gas-prices 5000000000atele --node %v --chain-id %v --keyring-backend test"+
				" --home ~/teleport_testnet/validators/validator0/teleport -y",
			encodedAddress, amountFaucet, node, chain)
		fmt.Println(time.Now().UTC().Format(time.RFC3339), encodedAddress, "[1]")
		executeCmd(sendFaucet, pass)

		time.Sleep(5 * time.Second)

		// sendSteak := fmt.Sprintf(
		// 	"gaiacli send --to=%v --name=%v --chain-id=%v --amount=%v",
		// 	encodedAddress, key, chain, amountSteak)
		// fmt.Println(time.Now().UTC().Format(time.RFC3339), encodedAddress, "[2]")
		// executeCmd(sendSteak, pass)
	}

	return
}
