package main

import (
	"fmt"
	"log"
	"strings"

	"strconv"

	"github.com/chzyer/readline"
	"github.com/conejoninja/tesoro"
	"github.com/zserge/hid"
)

func main() {

	tesoro.URIToIdentity("http://example.com:1234/path/to/something")

	var c tesoro.Client

	numberDevices := 0
	hid.UsbWalk(func(device hid.Device) {
		info := device.Info()
		// TREZOR
		// 0x534c : 21324 vendor
		// 0x0001 : 1     product
		// KEEPKEY
		// 0x2b24 : 11044 vendor
		// 0x0001 : 1     product
		if info.Vendor == 21324 && info.Product == 1 {
			numberDevices++
			c.SetTransport(device)
		}
	})
	if numberDevices == 0 {
		fmt.Println("No TREZOR devices found, make sure your device is connected")
	} else {
		fmt.Printf("Found %d TREZOR devices connected\n", numberDevices)
		shell(c)
		defer c.CloseTransport()
	}
}

func shell(c tesoro.Client) {
	var str string
	var msgType uint16
	rl, err := readline.NewEx(&readline.Config{
		Prompt: ">",
	})
	if err != nil {
		panic(err)
	}

	defer rl.Close()
	log.SetOutput(rl.Stderr())

	for {
		line, err := rl.Readline()
		if err != nil {
			fmt.Println("ERR", err)
			break
		}
		args := strings.Split(line, " ")

		switch strings.ToLower(args[0]) {
		case "ping":
			if len(args) < 2 {
				fmt.Println("Missing parameters")
			} else {
				pinProtection := false
				buttonProtection := false
				passphraseProtection := false

				if len(args) >= 3 {
					if args[2] == "1" || args[2] == "true" {
						pinProtection = true
					}
				}
				if len(args) >= 4 {
					if args[3] == "1" || args[3] == "true" {
						passphraseProtection = true
					}
				}
				if len(args) >= 5 {
					if args[4] == "1" || args[4] == "true" {
						buttonProtection = true
					}
				}

				str, msgType = c.Call(c.Ping(args[1], pinProtection, passphraseProtection, buttonProtection))
			}
			break
		case "signmessage":
			if len(args) < 2 {
				fmt.Println("Missing parameters")
			} else {
				msg := strings.Join(args[1:], " ")
				str, msgType = c.Call(c.SignMessage([]byte(msg)))
			}
			break
		case "verifymessage":
			if len(args) < 4 {
				fmt.Println("Missing parameters")
			} else {
				str, msgType = c.Call(c.VerifyMessage(args[1], args[2], []byte(args[3])))
			}
			break
		case "getaddress":
			var path string
			showDisplay := false
			coinName := "Bitcoin"
			if len(args) < 2 {
				path = "m/44'/0'/0'"
			} else {
				path = args[1]
			}
			if len(args) >= 3 {
				if args[2] == "1" || args[2] == "true" {
					showDisplay = true
				}
			}
			if len(args) >= 4 {
				coinName = args[3]
			}

			str, msgType = c.Call(c.GetAddress(tesoro.StringToBIP32Path(path), showDisplay, coinName))
			break
		case "getentropy":
			if len(args) < 2 {
				fmt.Println("Missing parameters")
			} else {
				size, _ := strconv.Atoi(args[1])
				str, msgType = c.Call(c.GetEntropy(uint32(size)))
			}
			break
		case "setlabel":
			if len(args) < 2 {
				fmt.Println("Missing parameters")
			} else {
				str, msgType = c.Call(c.SetLabel(strings.Join(args[1:], " ")))
			}
			break
		case "sethomescreen":
			if len(args) < 2 {
				fmt.Println("Missing parameters")
			} else {
				homescreen, err := tesoro.PNGToString(args[1])
				if err != nil {
					fmt.Println("Error reading image")
				} else {
					str, msgType = c.Call(c.SetHomescreen(homescreen))
				}
			}
			break
		case "getpublickey":
			var path string
			if len(args) < 2 {
				path = "m/44'/0'/0'"
			} else {
				path = args[1]
			}

			if !tesoro.ValidBIP32(path) {
				fmt.Println("Invalid BIP32 path. Example: m/44'/0'/0'/0/27 ")
			} else {
				str, msgType = c.Call(c.GetPublicKey(tesoro.StringToBIP32Path(path)))
			}
			break
		case "signidentity":
			var index uint32
			if len(args) < 4 {
				fmt.Println("Missing parameters")
			} else {
				if len(args) >= 5 {
					i, _ := strconv.Atoi(args[4])
					index = uint32(i)
				}
				str, msgType = c.Call(c.SignIdentity(args[1], []byte(args[2]), args[3], index))
			}
			break
		case "getfeatures":
			str, msgType = c.Call(c.GetFeatures())
			break
		case "clearsession":
			str, msgType = c.Call(c.ClearSession())
			break
		case "changepin":
			str, msgType = c.Call(c.ChangePin())
			break
		case "cipherkeyvalue":
			var path string
			var iv []byte
			encrypt := true
			askOnEncode := true
			askOnDecode := true
			if len(args) < 4 {
				fmt.Println("Missing parameters")
			} else {
				if args[1] == "0" || args[1] == "false" {
					encrypt = false
				}
				if len(args) < 5 {
					path = "m/44'/0'/0'"
				} else {
					path = args[4]
				}
				if len(args) >= 6 {
					iv = []byte(args[5])
				}
				if len(args) >= 7 && (args[6] == "0" || args[6] == "false") {
					askOnEncode = false
				}
				if len(args) >= 8 && (args[7] == "0" || args[7] == "false") {
					askOnDecode = false
				}
				if !tesoro.ValidBIP32(path) {
					fmt.Println("Invalid BIP32 path. Example: m/44'/0'/0'/0/27 ")
				} else {
					str, msgType = c.Call(c.CipherKeyValue(encrypt, args[2], []byte(args[3]), tesoro.StringToBIP32Path(path), iv, askOnEncode, askOnDecode))
				}
			}
			break
		default:
			if msgType == 18 { // PIN INPUT
				str, msgType = c.Call(c.PinMatrixAck(line))
			} else {
				fmt.Println("Unknown command")
				str = line
				msgType = 999
			}
			break
		}
		fmt.Println(str, msgType)
		if msgType == 26 {
			str, msgType = c.Call(c.ButtonAck())
			fmt.Println(str, msgType)
		}
	}
}
