package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"lmao/contracts"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

type AutoGenerated []struct {
	ID               string `json:"id"`
	TokenID          string `json:"tokenId"`
	Collection       string `json:"collection"`
	CollectionName   string `json:"collectionName"`
	CollectionSymbol string `json:"collectionSymbol"`
	CollectionSlug   string `json:"collectionSlug"`
	NumOwners        int    `json:"numOwners"`
	Metadata         struct {
		TokenURI   string `json:"tokenUri"`
		Attributes []struct {
			DisplayType     interface{} `json:"displayType"`
			TraitType       string      `json:"traitType"`
			Value           string      `json:"value"`
			Count           interface{} `json:"count"`
			CountPercentage interface{} `json:"countPercentage"`
			RarityScore     interface{} `json:"rarityScore"`
		} `json:"attributes"`
		Description  string      `json:"description"`
		ExternalURL  interface{} `json:"externalUrl"`
		Image        string      `json:"image"`
		AnimationURL interface{} `json:"animationUrl"`
		Name         string      `json:"name"`
	} `json:"metadata"`
	RarityScore   interface{} `json:"rarityScore"`
	RarityRanking int         `json:"rarityRanking"`
	Verified      string      `json:"verified"`
	BestBid       struct {
		IsOrderAsk         bool   `json:"isOrderAsk"`
		Signer             string `json:"signer"`
		Collection         string `json:"collection"`
		Strategy           string `json:"strategy"`
		Currency           string `json:"currency"`
		Params             string `json:"params"`
		Price              string `json:"price"`
		TokenID            string `json:"tokenId"`
		Amount             string `json:"amount"`
		Nonce              string `json:"nonce"`
		StartTime          string `json:"startTime"`
		EndTime            string `json:"endTime"`
		MinPercentageToAsk string `json:"minPercentageToAsk"`
		V                  int    `json:"v"`
		R                  string `json:"r"`
		S                  string `json:"s"`
	} `json:"bestBid"`
	CurrentAsk struct {
		IsOrderAsk         bool   `json:"isOrderAsk"`
		Signer             string `json:"signer"`
		Collection         string `json:"collection"`
		Strategy           string `json:"strategy"`
		Currency           string `json:"currency"`
		Params             string `json:"params"`
		Price              string `json:"price"`
		TokenID            string `json:"tokenId"`
		Amount             string `json:"amount"`
		Nonce              string `json:"nonce"`
		StartTime          string `json:"startTime"`
		EndTime            string `json:"endTime"`
		MinPercentageToAsk string `json:"minPercentageToAsk"`
		V                  int    `json:"v"`
		R                  string `json:"r"`
		S                  string `json:"s"`
	} `json:"currentAsk"`
	Owner struct {
		ID       string `json:"id"`
		OwnerID  string `json:"ownerId"`
		Quantity int    `json:"quantity"`
	} `json:"owner"`
}

func PrettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}
func main() {
	log.Printf("started.")

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	client, err := ethclient.Dial(os.Getenv("AVAX_RPC"))
	if err != nil {
		log.Fatal(err)
	}
	privateKey, err := crypto.HexToECDSA(os.Getenv("PRIVATE_KEY_JOEPEG"))
	if err != nil {
		log.Fatal(err)
	}
	// fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	joepegContract, err := contracts.NewJoepeg(common.HexToAddress(os.Getenv("CONTRACT_JOEPEG")), client)
	if err != nil {
		log.Fatalf("Failed to instantiate a Joepeg contract: %v", err)
	}

	for {
		log.Printf("running.")
		func() {
			//发送请求获取响应
			resp, err := http.Get("https://barn.joepegs.com/v3/items/?pageSize=20&pageNum=1&orderBy=recent_listing&filters=has_offers&chain=avalanche")
			//结束网络释放资源
			if resp != nil {
				defer resp.Body.Close()
			}
			if err != nil {
				log.Println(err)
				return
			}

			//判断响应状态码
			if resp.StatusCode != http.StatusOK {
				log.Println("Error StatusCode")
				return
			}
			body, _ := ioutil.ReadAll(resp.Body)

			//读取响应实体
			var result AutoGenerated
			if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
				log.Println("Can not unmarshal JSON")
			}
			// log.Println(result)

			for _, v := range result {
				buyPrice, _ := new(big.Int).SetString(v.CurrentAsk.Price, 10)
				sellPrice, _ := new(big.Int).SetString(v.BestBid.Price, 10)
				if buyPrice == nil || sellPrice == nil {
					continue
				}
				// log.Println(buyPrice)
				// log.Println(sellPrice)
				if buyPrice.Cmp(sellPrice) < 0 {
					// 调用套利合约
					auth, _ := bind.NewKeyedTransactorWithChainID(privateKey, chainID)

					orderPrice := buyPrice
					tokenId, _ := new(big.Int).SetString(v.CurrentAsk.TokenID, 10)
					amount, _ := new(big.Int).SetString(v.CurrentAsk.Amount, 10)
					nonce, _ := new(big.Int).SetString(v.CurrentAsk.Nonce, 10)
					startTime, _ := new(big.Int).SetString(v.CurrentAsk.StartTime, 10)
					endTime, _ := new(big.Int).SetString(v.CurrentAsk.EndTime, 10)
					minPercentageToAsk, _ := new(big.Int).SetString(v.CurrentAsk.MinPercentageToAsk, 10)

					R := [32]byte{}
					S := [32]byte{}
					copy(R[:], common.FromHex(v.CurrentAsk.R))
					copy(S[:], common.FromHex(v.CurrentAsk.S))
					order := contracts.MakerOrder{
						IsOrderAsk:         v.CurrentAsk.IsOrderAsk,
						Signer:             common.HexToAddress(v.CurrentAsk.Signer),
						Collection:         common.HexToAddress(v.CurrentAsk.Collection),
						Price:              orderPrice,
						TokenId:            tokenId,
						Amount:             amount,
						Strategy:           common.HexToAddress(v.CurrentAsk.Strategy),
						Currency:           common.HexToAddress(v.CurrentAsk.Currency),
						Nonce:              nonce,
						StartTime:          startTime,
						EndTime:            endTime,
						MinPercentageToAsk: minPercentageToAsk,
						Params:             []byte{},
						V:                  uint8(v.CurrentAsk.V),
						R:                  R,
						S:                  S,
					}

					offerOrderPrice := sellPrice
					offerTokenId, _ := new(big.Int).SetString(v.BestBid.TokenID, 10)
					offerAmount, _ := new(big.Int).SetString(v.BestBid.Amount, 10)
					offerNonce, _ := new(big.Int).SetString(v.BestBid.Nonce, 10)
					offerStartTime, _ := new(big.Int).SetString(v.BestBid.StartTime, 10)
					offerEndTime, _ := new(big.Int).SetString(v.BestBid.EndTime, 10)
					offerMinPercentageToAsk, _ := new(big.Int).SetString(v.BestBid.MinPercentageToAsk, 10)

					offerR := [32]byte{}
					offerS := [32]byte{}
					copy(offerR[:], common.FromHex(v.BestBid.R))
					copy(offerS[:], common.FromHex(v.BestBid.S))
					offerOrder := contracts.MakerOrder{
						IsOrderAsk:         v.BestBid.IsOrderAsk,
						Signer:             common.HexToAddress(v.BestBid.Signer),
						Collection:         common.HexToAddress(v.BestBid.Collection),
						Price:              offerOrderPrice,
						TokenId:            offerTokenId,
						Amount:             offerAmount,
						Strategy:           common.HexToAddress(v.BestBid.Strategy),
						Currency:           common.HexToAddress(v.BestBid.Currency),
						Nonce:              offerNonce,
						StartTime:          offerStartTime,
						EndTime:            offerEndTime,
						MinPercentageToAsk: offerMinPercentageToAsk,
						Params:             []byte{},
						V:                  uint8(v.BestBid.V),
						R:                  offerR,
						S:                  offerS,
					}
					signedTx, err := joepegContract.Go(auth, order, offerOrder, big.NewInt(0))

					if err != nil {
						log.Println(err)
						continue
					}
					log.Printf("Joepeg commited. TX: [%s]", signedTx.Hash().Hex())
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
					defer cancel()
					receipt, err := bind.WaitMined(ctx, client, signedTx)
					if err != nil {
						log.Println(err)
						continue
					}
					log.Printf("Joepeg successed. TX: [%s]", receipt.TxHash.Hex())
				}
			}
		}()

		time.Sleep(time.Second)
	}

}
