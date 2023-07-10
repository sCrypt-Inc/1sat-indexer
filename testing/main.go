package main

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/libsv/go-bt/v2"
	"github.com/ordishs/go-bitcoin"
	"github.com/redis/go-redis/v9"
	"github.com/shruggr/1sat-indexer/lib"
)

var rdb *redis.Client
var bit *bitcoin.Bitcoind

func main() {
	godotenv.Load("../.env")

	db, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES"))
	if err != nil {
		log.Panic(err)
	}

	port, _ := strconv.ParseInt(os.Getenv("BITCOIN_PORT"), 10, 32)
	bit, err = bitcoin.New(os.Getenv("BITCOIN_HOST"), int(port), os.Getenv("BITCOIN_USER"), os.Getenv("BITCOIN_PASS"), false)
	if err != nil {
		log.Panic(err)
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	err = lib.Initialize(db, rdb)
	if err != nil {
		log.Panic(err)
	}
	hexId := "e17d7856c375640427943395d2341b6ed75f73afc8b22bb3681987278978a584"
	txid, _ := hex.DecodeString(hexId)

	tx := bt.NewTx()
	r, err := bit.GetRawTransactionRest(hexId)
	if err != nil {
		log.Panicf("%x: %v\n", txid, err)
	}
	if _, err = tx.ReadFrom(r); err != nil {
		log.Panicf("%x: %v\n", txid, err)
	}
	height := uint32(783968)
	result, err := lib.IndexTxn(tx, nil, &height, 0, false)
	if err != nil {
		log.Panic(err)
	}

	log.Println("Origins:", len(result.Origins))

	// txbuf, err := hex.DecodeString("0100000001e5c475770dd6b98d12abfc61452178d986de20d88ff2adfc1ec1f0d287aa81a9000000006a47304402203274dcb66c12f437d3fe66d5a71f2e0a3d9599ee1ee42beb7504c2f438102a7702201e2cd42faef2ff34186d878af36a63fd9264670773489b9892230a3e8e68b6e4412103b5dd4f7293b014689d7b944a0c7104d82b51f3db871cfbb75b944f908410c845ffffffff020100000000000000fdc60776a9149be4820c478ce90d66ed4fa2df4ef4cea5a8b17888ac0063036f72645109696d6167652f706e67004dad0689504e470d0a1a0a0000000d494844520000019000000190040300000072912bff0000000f504c5445556c8e556c8e556c8e556c8e566d8fa7a4a1fd0000000474524e53002a81d2cc0cbf5e000006494944415478daed9d5b72db38140509700370c805c8132e40b2b98091c4fdaf2965d9165f00a8c72441df39c77faea4caad16403c2ec0aa52144551144551144551144551a671c108487d3202d20d3b131c7e18ce46840cc3c142531f3e62a0bdb7171003edbdbf80f0db7bfdc9c16fefdd1708bdbdfb6f0eba9276b061c45d39e01d707de580f7bffd1564a7a65e54df3bec8d34f533bba98f7def912da457df5b6a5357dffb679ac0fb57f6f0bed7e7ff4ece306b0384d3f76e80708659791050df9b07010db3b220a4716f168434ccca81a0a6b83910d4302b07829ae266403c6adc9b01614d71d3209e35c54d83c0a6b84910da14d7a5406853dca411daf262ca086e79316504b7bc9830c2db494818e12d2f268cf07612e24680cb8b7123c09d84a811e2d64ed4087127210682dcc58d7db51ae24e42cc0872173702e2915b3b1110e62eee1ac4337771d720d05ddc2bc8bfd0296ed20875177765845a41b33482ada0591ac156d02c8c708be51646b815340b23dc62b9b9117005cddc08b8586e66845cbd3833422e549e1a41172a4f411a72b1dc14045da83c01f1e8eac50908bb50790481578e8f20f042e52bc8017e48e40a7286172a4f1e82ec42e53508f490c81a646f05241801a11e1271369afa1a047b48c4d968ea2b10ee01bd05c8c10a483002023ea0370739bc64821b34a62210810844200211884004221081084420ff2dc8fb98fe122848262184f0e22833444551943f981fff3c9e9238fcf0444a5aaeee9f012968bdfa2921252d58b7cf8194b3c9d63d0772b602329801093222233222233222233222233222235823e7b7f7de8291d75055ee27dfc8ebe72f1bba91e3fd2da74823e33d139e6de4f8406756a491c9a75b938d4cd7a81cd9c8f19107658946768fac40966864f6e1d660230f2d6e176864be8cebb8464e0f8154a5835432a236f277418a7f8e782b4ff61a6c64ffc810a54423b341630f36727e64185fe40c7177ff37abcc19e2e9fe6f56a1ab2857250d7b15e5ba8c72471d6fa12b8da7cbdfe5ee58342d75edf7fc1a5e7edcb3f8abd5f8d2406444466444464a06f969c4c871fdcf9946c27a2a8f34728c8c8b91464264404934728cad13118d84d8ff001a3946d725804642741ecc3392d884e71909f175619c9163623d156764f6c997f8568eee0121d32f57b9464efd2dd38eae7c23bbf6963361ae782387e832e3fa736f0a37f2b168dade7448af7023bb9baf582d7ced77f998489f9a24acc6bb5b6681451bd9c786858963ac881d2b77c39483b187d86e9f2b2ed8c821f2bc4b7fe6905a9466f3a077b94676f1f16da860464ef12148f2e47da94696ef5ffc56122a98917d7c5498be0aa15023eb2b0f3e95840a662472d17b93bf9ba2502311900f25a13260a46ab29785708c54ae0f95092355f69e3590917c48462a199111199111199111199111199111199111199191ff8f11a74aec42416444466444466444469e4839b794d7cf8194736fbc7b0ea4a0b765b73684dc79bde4b02ee52c28e1e5d19f4a511445514a8dbbf96d5aaf658378e0105c2002118840042210810844200211880d90af89d5dbdbdbfb3a3d08249bd608881b8c80d456407a23207e3002d2190171831190d608c8ea82682a483d1801e98d80f8c10848670464ec7bcf1d1a64ec7b8f2d1a646cea3bb4917a52078306e92637369141fcb4ab2283b4d3ab80c0206e56150a06a967256f60907e5683c805f1f3510917a49bdf968705f18b624a2c48bbb8868d0ae2fa45b93175d0582fcb8da9469642a846fc6a460835d2ad6e2a651a71ebc3374c23edfa0a4ca6917e7dd40369a48e1cf5401ae922ef68251af1b1d538a2913676db2d10c4450f3e02bf5a4df45417d0481f3d66c703f1f185771e48977f3ff6192824a041dac44de33410973a51db167784fbfe6116d2489f3ae20c03f1c94d4f1848a2efc581b8f4fd12acc6dea6dff2c032d2a76f334019a933b719a08ca49b3acb88cf159c908cb4b917ba808cb8ecdd3e20234df6e21290913e7b930c07c4e76bcb3820b9be9704e237aec4c180b41b2fd3a280b8ad4ba32820f5d6a55114907eeb162fff7d7b05baefe5a48bbd16101857e0158abfa5efc564b3a943b2d9f7f29afa1ecd61a6efb5d2d4cdf4bd8d95a66ea5ef1dcb30e14d7d7c8aece120d7c748a0837813c3acf141b2e3835cdafba93290c68690cbb3e46c82a3f2f029eea40b0e95a2288aa2288aa2288aa2287f33bf0086878c5a39447e940000000049454e44ae426082686a223150755161374b36324d694b43747373534c4b79316b683536575755374d7455523503534554036170700361796d0474797065036f7264046e616d650954455354206e616d650b7375625479706544617461227b226465736372697074696f6e223a2254455354206465736372697074696f6e227d017c055349474d410342534d2231464448556b4e7535514c48315868646a4a337470634556536574423551686e435a411feb9d9d731a634bd19d4ad1cc9e76957514699ddc553e7895be5080038f38380c0e9b46baeae5626623df9266b7ab16e7327fed750381aa81bea3cf425587cb140130a0252600000000001976a91468a78d9b759b64c3e43ce0e3d5f477302024df1a88ac00000000")
	// // txbuf, err := hex.DecodeString("0100000001583498a165c40edc029fd5af95f056a23cc3fcdf1f72ef96ab587ebdb5197d44010000006a4730440220777f152557fe39f2110ad78556ad5a7cadb19df1932a35c505261a7555288fe1022049c8ca289c1d5d48b6ed2cfd114edd1bc81f4425aeb6881aae135417e0ccdd2b412102f06754229ba26b8f3b4aedf3a40dfd2885f9b59ce522e2caad850cbcdb731a0effffffff020100000000000000fd4e0776a914e0a630d5395b510c5ce3647b12cafe2c9dc8b1a988ac0063036f72645109696d6167652f706e67004dad0689504e470d0a1a0a0000000d494844520000019000000190040300000072912bff0000000f504c5445556c8e556c8e556c8e556c8e566d8fa7a4a1fd0000000474524e53002a81d2cc0cbf5e000006494944415478daed9d5b72db38140509700370c805c8132e40b2b98091c4fdaf2965d9165f00a8c72441df39c77faea4caad16403c2ec0aa52144551144551144551144551a671c108487d3202d20d3b131c7e18ce46840cc3c142531f3e62a0bdb7171003edbdbf80f0db7bfdc9c16fefdd1708bdbdfb6f0eba9276b061c45d39e01d707de580f7bffd1564a7a65e54df3bec8d34f533bba98f7def912da457df5b6a5357dffb679ac0fb57f6f0bed7e7ff4ece306b0384d3f76e80708659791050df9b07010db3b220a4716f168434ccca81a0a6b83910d4302b07829ae266403c6adc9b01614d71d3209e35c54d83c0a6b84910da14d7a5406853dca411daf262ca086e79316504b7bc9830c2db494818e12d2f268cf07612e24680cb8b7123c09d84a811e2d64ed4087127210682dcc58d7db51ae24e42cc0872173702e2915b3b1110e62eee1ac4337771d720d05ddc2bc8bfd0296ed20875177765845a41b33482ada0591ac156d02c8c708be51646b815340b23dc62b9b9117005cddc08b8586e66845cbd3833422e549e1a41172a4f411a72b1dc14045da83c01f1e8eac50908bb50790481578e8f20f042e52bc8017e48e40a7286172a4f1e82ec42e53508f490c81a646f05241801a11e1271369afa1a047b48c4d968ea2b10ee01bd05c8c10a483002023ea0370739bc64821b34a62210810844200211884004221081084420ff2dc8fb98fe122848262184f0e22833444551943f981fff3c9e9238fcf0444a5aaeee9f012968bdfa2921252d58b7cf8194b3c9d63d0772b602329801093222233222233222233222233222235823e7b7f7de8291d75055ee27dfc8ebe72f1bba91e3fd2da74823e33d139e6de4f8406756a491c9a75b938d4cd7a81cd9c8f19107658946768fac40966864f6e1d660230f2d6e176864be8cebb8464e0f8154a5835432a236f277418a7f8e782b4ff61a6c64ffc810a54423b341630f36727e64185fe40c7177ff37abcc19e2e9fe6f56a1ab2857250d7b15e5ba8c72471d6fa12b8da7cbdfe5ee58342d75edf7fc1a5e7edcb3f8abd5f8d2406444466444464a06f969c4c871fdcf9946c27a2a8f34728c8c8b91464264404934728cad13118d84d8ff001a3946d725804642741ecc3392d884e71909f175619c9163623d156764f6c997f8568eee0121d32f57b9464efd2dd38eae7c23bbf6963361ae782387e832e3fa736f0a37f2b168dade7448af7023bb9baf582d7ced77f998489f9a24acc6bb5b6681451bd9c786858963ac881d2b77c39483b187d86e9f2b2ed8c821f2bc4b7fe6905a9466f3a077b94676f1f16da860464ef12148f2e47da94696ef5ffc56122a98917d7c5498be0aa15023eb2b0f3e95840a662472d17b93bf9ba2502311900f25a13260a46ab29785708c54ae0f95092355f69e3590917c48462a199111199111199111199111199111199111199191ff8f11a74aec42416444466444466444469e4839b794d7cf8194736fbc7b0ea4a0b765b73684dc79bde4b02ee52c28e1e5d19f4a511445514a8dbbf96d5aaf658378e0105c2002118840042210810844200211880d90af89d5dbdbdbfb3a3d08249bd608881b8c80d456407a23207e3002d2190171831190d608c8ea82682a483d1801e98d80f8c10848670464ec7bcf1d1a64ec7b8f2d1a646cea3bb4917a52078306e92637369141fcb4ab2283b4d3ab80c0206e56150a06a967256f60907e5683c805f1f3510917a49bdf968705f18b624a2c48bbb8868d0ae2fa45b93175d0582fcb8da9469642a846fc6a460835d2ad6e2a651a71ebc3374c23edfa0a4ca6917e7dd40369a48e1cf5401ae922ef68251af1b1d538a2913676db2d10c4450f3e02bf5a4df45417d0481f3d66c703f1f185771e48977f3ff6192824a041dac44de33410973a51db167784fbfe6116d2489f3ae20c03f1c94d4f1848a2efc581b8f4fd12acc6dea6dff2c032d2a76f334019a933b719a08ca49b3acb88cf159c908cb4b917ba808cb8ecdd3e20234df6e21290913e7b930c07c4e76bcb3820b9be9704e237aec4c180b41b2fd3a280b8ad4ba32820f5d6a55114907eeb162fff7d7b05baefe5a48bbd16101857e0158abfa5efc564b3a943b2d9f7f29afa1ecd61a6efb5d2d4cdf4bd8d95a66ea5ef1dcb30e14d7d7c8aece120d7c748a0837813c3acf141b2e3835cdafba93290c68690cbb3e46c82a3f2f029eea40b0e95a2288aa2288aa2288aa2287f33bf0086878c5a39447e940000000049454e44ae426082686a055349474d410342534d22314535533931716e6f4743586d36314d5931617842435a436d4d50414d5a3675457a4120bee25c0b037caafade65dc0c71b7a3c8039732664874dcdd1cc5c380bd255fa24c032edff9e9b1477c1d25a86062be008e2f1ddaca81380aa6a09693d84b6a770130ff2e0400000000001976a914862edde3cbaf3487c169eb253737c89059dda0b388ac00000000")
	// if err != nil {
	// 	log.Panic(err)
	// }
	// tx, err := bt.NewTxFromBytes(txbuf)
	// if err != nil {
	// 	log.Panic(err)
	// }
	// result, err := lib.IndexTxn(tx, 0, 0, true)

	// out, err := json.MarshalIndent(result, "", "  ")
	// if err != nil {
	// 	log.Panic(err)
	// }
	// fmt.Println(string(out))
}
