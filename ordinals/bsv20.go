package ordinals

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/libsv/go-bk/bip32"
	"github.com/libsv/go-bk/crypto"
	"github.com/libsv/go-bt/bscript"
	"github.com/shruggr/1sat-indexer/lib"
	"github.com/shruggr/1sat-indexer/ordlock"
)

type Bsv20Status int

var hdKey, _ = bip32.NewKeyFromString("xpub661MyMwAqRbcF221R74MPqdipLsgUevAAX4hZP2rywyEeShpbe3v2r9ciAvSGT6FB22TEmFLdUyeEDJL4ekG8s9H5WXbzDQPr6eW1zEYYy9")

const (
	Invalid Bsv20Status = -1
	Pending Bsv20Status = 0
	Valid   Bsv20Status = 1
	MintFee uint64      = 100
)

const BSV20V1_OP_COST = 1000
const BSV20V2_OP_COST = 1000

var ctx = context.Background()

type Bsv20 struct {
	Txid          []byte        `json:"-"`
	Vout          uint32        `json:"-"`
	Height        *uint32       `json:"-"`
	Idx           uint64        `json:"-"`
	Ticker        string        `json:"tick,omitempty"`
	Id            *lib.Outpoint `json:"id,omitempty"`
	Op            string        `json:"op"`
	Symbol        *string       `json:"sym,omitempty"`
	Max           uint64        `json:"-"`
	Limit         uint64        `json:"-"`
	Decimals      uint8         `json:"dec,omitempty"`
	Icon          *lib.Outpoint `json:"icon,omitempty"`
	Supply        uint64        `json:"-"`
	Amt           *uint64       `json:"amt"`
	Implied       bool          `json:"-,omitempty"`
	Status        Bsv20Status   `json:"-"`
	Reason        *string       `json:"reason,omitempty"`
	PKHash        []byte        `json:"-"`
	Price         uint64        `json:"-"`
	PayOut        []byte        `json:"-"`
	Listing       bool          `json:"-"`
	PricePerToken float64       `json:"-"`
	FundPath      string        `json:"-"`
	FundPKHash    []byte        `json:"-"`
	FundBalance   int           `json:"-"`
}

func ParseBsv20(ctx *lib.IndexContext) {
	for _, txo := range ctx.Txos {
		if bsv20, ok := txo.Data["bsv20"].(*Bsv20); ok {
			list := ordlock.ParseScript(txo)

			if list != nil {
				txo.PKHash = list.PKHash
				bsv20.PKHash = list.PKHash
				bsv20.Price = list.Price
				bsv20.PayOut = list.PayOut
				bsv20.Listing = true

				var decimals uint8
				if bsv20.Ticker != "" {
					token := LoadTicker(bsv20.Ticker)
					if token != nil {
						decimals = token.Decimals
					}
				} else {
					token := LoadTokenById(bsv20.Id)
					if token != nil {
						decimals = token.Decimals
					}
				}
				bsv20.PricePerToken = float64(bsv20.Price) / float64(*bsv20.Amt) * float64(10^uint64(decimals))
			}
		}
	}
}

func IndexBsv20(ctx *lib.IndexContext) {
	ParseBsv20(ctx)
	for _, txo := range ctx.Txos {
		if bsv20, ok := txo.Data["bsv20"].(*Bsv20); ok {
			bsv20.Save(txo)
		}
	}
}

func ParseBsv20Inscription(ord *lib.File, txo *lib.Txo) (bsv20 *Bsv20, err error) {
	mime := strings.ToLower(ord.Type)
	if !strings.HasPrefix(mime, "application/bsv-20") &&
		!(txo.Height != nil && *txo.Height < 793000 && strings.HasPrefix(mime, "text/plain")) {
		return nil, nil
	}
	data := map[string]string{}
	err = json.Unmarshal(ord.Content, &data)
	if err != nil {
		// fmt.Println("JSON PARSE ERROR:", ord.Content, err)
		return
	}
	var protocol string
	var ok bool
	if protocol, ok = data["p"]; !ok || protocol != "bsv-20" {
		return nil, nil
	}
	bsv20 = &Bsv20{}
	if op, ok := data["op"]; ok {
		bsv20.Op = strings.ToLower(op)
	} else {
		return nil, nil
	}

	if tick, ok := data["tick"]; ok {
		chars := []rune(tick)
		if len(chars) > 4 {
			return nil, nil
		}
		tick = strings.ToUpper(tick)
		bsv20.Ticker = tick
	}

	if id, ok := data["id"]; ok && len(id) >= 66 {
		bsv20.Id, err = lib.NewOutpointFromString(id)
		if err != nil {
			return nil, nil
		}
	}

	if amt, ok := data["amt"]; ok {
		amt, err := strconv.ParseUint(amt, 10, 64)
		if err != nil {
			return nil, nil
		}
		bsv20.Amt = &amt
	}

	if dec, ok := data["dec"]; ok {
		var val uint64
		val, err = strconv.ParseUint(dec, 10, 8)
		if err != nil || val > 18 {
			return nil, nil
		}
		bsv20.Decimals = uint8(val)
	}

	switch bsv20.Op {
	case "deploy":
		if max, ok := data["max"]; ok {
			bsv20.Max, err = strconv.ParseUint(max, 10, 64)
			if err != nil {
				return nil, nil
			}
		}
		if limit, ok := data["lim"]; ok {
			bsv20.Limit, err = strconv.ParseUint(limit, 10, 64)
			if err != nil {
				return nil, nil
			}
		}
		hash := sha256.Sum256([]byte(bsv20.Ticker))
		path := fmt.Sprintf("21/%d/%d", binary.BigEndian.Uint32(hash[:8])>>1, binary.BigEndian.Uint32(hash[24:])>>1)
		ek, err := hdKey.DeriveChildFromPath(path)
		if err != nil {
			log.Panic(err)
		}
		pubKey, err := ek.ECPubKey()
		if err != nil {
			log.Panic(err)
		}
		bsv20.FundPath = path
		bsv20.FundPKHash = crypto.Hash160(pubKey.SerialiseCompressed())
	case "deploy+mint":
		if bsv20.Amt == nil {
			return nil, nil
		}
		if sym, ok := data["sym"]; ok {
			bsv20.Symbol = &sym
		}
		bsv20.Ticker = ""
		bsv20.Status = Valid
		if icon, ok := data["icon"]; ok {
			bsv20.Icon, _ = lib.NewOutpointFromString(icon)
		}
		bsv20.Id = txo.Outpoint
		hash := sha256.Sum256(*bsv20.Id)
		path := fmt.Sprintf("21/%d/%d", binary.BigEndian.Uint32(hash[:8])>>1, binary.BigEndian.Uint32(hash[24:])>>1)
		ek, err := hdKey.DeriveChildFromPath(path)
		if err != nil {
			log.Panic(err)
		}
		pubKey, err := ek.ECPubKey()
		if err != nil {
			log.Panic(err)
		}
		bsv20.FundPath = path
		bsv20.FundPKHash = crypto.Hash160(pubKey.SerialiseCompressed())
	case "mint":
		if bsv20.Ticker == "" || bsv20.Amt == nil {
			return nil, nil
		}
	case "transfer":
		if bsv20.Amt == nil {
			return nil, nil
		}
		if bsv20.Ticker == "" && bsv20.Id == nil {
			return nil, nil
		}
	default:
		return nil, nil
	}

	return bsv20, nil
}

func (b *Bsv20) Save(t *lib.Txo) {
	if b.Op == "deploy" {
		_, err := Db.Exec(ctx, `
			INSERT INTO bsv20(txid, vout, height, idx, tick, max, lim, dec, status, reason, fund_path, fund_pkhash)
			VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT(txid, vout) DO UPDATE SET
				height=EXCLUDED.height,
				idx=EXCLUDED.idx,
				fund_path=EXCLUDED.fund_path,
				fund_pkhash=EXCLUDED.fund_pkhash`,
			t.Outpoint.Txid(),
			t.Outpoint.Vout(),
			t.Height,
			t.Idx,
			b.Ticker,
			strconv.FormatUint(b.Max, 10),
			b.Limit,
			b.Decimals,
			b.Status,
			b.Reason,
			b.FundPath,
			b.FundPKHash,
		)
		if err != nil {
			log.Panic(err)
		}
	}
	if b.Op == "deploy+mint" {
		_, err := Db.Exec(ctx, `
			INSERT INTO bsv20_v2(id, height, idx, sym, icon, amt, dec, fund_path, fund_pkhash)
			VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT(id) DO UPDATE SET
				height=EXCLUDED.height,
				idx=EXCLUDED.idx,
				sym=EXCLUDED.sym,
				icon=EXCLUDED.icon,
				fund_path=EXCLUDED.fund_path,
				fund_pkhash=EXCLUDED.fund_pkhash`,
			t.Outpoint,
			t.Height,
			t.Idx,
			b.Symbol,
			b.Icon,
			b.Amt,
			b.Decimals,
			b.FundPath,
			b.FundPKHash,
		)
		if err != nil {
			log.Panic(err)
		}
	}
	if b.Op == "deploy+mint" || b.Op == "mint" || b.Op == "transfer" {
		// log.Println("BSV20 TXO:", b.Ticker, b.Id)

		// log.Printf("BSV20 TXO: %s %d\n", b.Id, len(t.Script))
		_, err := Db.Exec(ctx, `
			INSERT INTO bsv20_txos(txid, vout, height, idx, id, tick, op, amt, pkhash, price, payout, listing, price_per_token, script, status, spend)
			SELECT txid, vout, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, spend
			FROM txos
			WHERE txid=$1 AND vout=$2
			ON CONFLICT(txid, vout) DO UPDATE SET
				height=EXCLUDED.height,
				idx=EXCLUDED.idx,
				pkhash=EXCLUDED.pkhash,
				status=EXCLUDED.status,
				script=EXCLUDED.script,
				price=EXCLUDED.price,
				payout=EXCLUDED.payout,
				listing=EXCLUDED.listing,
				price_per_token=EXCLUDED.price_per_token`,
			t.Outpoint.Txid(),
			t.Outpoint.Vout(),
			t.Height,
			t.Idx,
			b.Id,
			b.Ticker,
			b.Op,
			b.Amt,
			t.PKHash,
			b.Price,
			b.PayOut,
			b.Listing,
			b.PricePerToken,
			t.Script,
			b.Status,
		)

		if err != nil {
			log.Panicf("%x %v", t.Outpoint.Txid(), err)
		}
	}
}

func ValidateBsv20Deploy(height uint32) {
	rows, err := Db.Query(ctx, `
		SELECT txid, vout, height, idx, tick, max, lim, supply
		FROM bsv20
		WHERE status=0 AND height <= $1 AND height IS NOT NULL
		ORDER BY height ASC, idx ASC, vout ASC`,
		height,
	)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		ticker := &Bsv20{}
		err = rows.Scan(&ticker.Txid, &ticker.Vout, &ticker.Height, &ticker.Idx, &ticker.Ticker, &ticker.Max, &ticker.Limit, &ticker.Supply)
		if err != nil {
			log.Panic(err)
		}

		func(bsv20 *Bsv20) {
			rows, err := Db.Query(ctx, `
				SELECT txid, vout
				FROM bsv20
				WHERE tick=$1 AND status=1 AND (
					height < $2 OR
					(height = $2 AND idx < $3) OR 
					(height = $2 AND idx = $3 AND vout < $4)
				)
				LIMIT 1`,
				bsv20.Ticker,
				bsv20.Height,
				bsv20.Idx,
				bsv20.Vout,
			)
			if err != nil {
				log.Panic(err)
			}
			defer rows.Close()

			if rows.Next() {
				_, err = Db.Exec(ctx, `
					UPDATE bsv20
					SET status = -1, reason='duplicate'
					WHERE txid = $1 AND vout = $2`,
					bsv20.Txid,
					bsv20.Vout,
				)
				if err != nil {
					log.Panic(err)
				}
				return
			}
			_, err = Db.Exec(ctx, `
				UPDATE bsv20
				SET status = 1
				WHERE txid = $1 AND vout = $2`,
				bsv20.Txid,
				bsv20.Vout,
			)
			if err != nil {
				log.Panic(err)
			}
		}(ticker)
	}
}

func ValidateV1Transfer(txid []byte, tick string, mined bool) int {
	// log.Printf("Validating %x %s\n", txid, tick)
	balance, err := Rdb.Get(ctx, "funds:"+tick).Int64()
	if err != nil {
		panic(err)
	}
	if balance < BSV20V1_OP_COST {
		log.Printf("Inadequate funding %x %s\n", txid, tick)
		return 0
	}

	inRows, err := Db.Query(ctx, `
		SELECT txid, vout, status, amt
		FROM bsv20_txos
		WHERE spend=$1 AND tick=$2`,
		txid,
		tick,
	)
	if err != nil {
		log.Panicf("%x - %v\n", txid, err)
	}
	defer inRows.Close()

	var reason string
	var tokensIn uint64
	var tokenOuts []uint32
	for inRows.Next() {
		var inTxid []byte
		var vout uint32
		var amt uint64
		var inStatus int
		err = inRows.Scan(&inTxid, &vout, &inStatus, &amt)
		if err != nil {
			log.Panicf("%x - %v\n", txid, err)
		}

		switch inStatus {
		case -1:
			reason = "invalid input"
		case 0:
			fmt.Printf("inputs pending %s %x\n", tick, txid)
			return 0
		case 1:
			tokensIn += amt
		}
	}
	inRows.Close()

	sql := `SELECT vout, status, amt
		FROM bsv20_txos
		WHERE txid=$1 AND tick=$2 AND op='transfer'`
	outRows, err := Db.Query(ctx,
		sql,
		txid,
		tick,
	)
	if err != nil {
		log.Panicf("%x - %v\n", txid, err)
	}
	defer outRows.Close()

	for outRows.Next() {
		var vout uint32
		var amt uint64
		var status int
		err = outRows.Scan(&vout, &status, &amt)
		if err != nil {
			log.Panicf("%x - %v\n", txid, err)
		}
		tokenOuts = append(tokenOuts, vout)
		if amt > tokensIn {
			if reason == "" {
				reason = fmt.Sprintf("insufficient balance %d < %d", tokensIn, amt)
			}
			if !mined {
				fmt.Printf("%s %s - %x\n", tick, reason, txid)
				return 0
			}
		} else {
			tokensIn -= amt
		}
	}

	if reason != "" {
		log.Printf("Transfer Invalid: %x %s %s\n", txid, tick, reason)
		sql := `UPDATE bsv20_txos
			SET status=-1, reason=$3
			WHERE txid=$1 AND vout=ANY($2)`
		_, err := Db.Exec(ctx, sql,
			txid,
			tokenOuts,
			reason,
		)
		if err != nil {
			log.Panicf("%x %v\n", txid, err)
		}
	} else {
		log.Printf("Transfer Valid: %x %s\n", txid, tick)
		_, err := Db.Exec(ctx, `
				UPDATE bsv20_txos
				SET status=1
				WHERE txid=$1 AND vout=ANY ($2)`,
			txid,
			tokenOuts,
		)
		if err != nil {
			log.Panicf("%x %v\n", txid, err)
		}
	}
	Rdb.IncrBy(ctx, "funds:"+tick, int64(len(tokenOuts)*BSV20V1_OP_COST*-1))
	return len(tokenOuts)
}

func ValidateV2Transfer(txid []byte, id *lib.Outpoint, mined bool) (outputs int) {
	// log.Printf("Validating V2 Transfer %x %s\n", txid, id.String())
	balance, err := Rdb.Get(ctx, "funds:"+id.String()).Int64()
	if err != nil {
		panic(err)
	}
	if balance < BSV20V2_OP_COST {
		log.Printf("Inadequate funding %x %s\n", txid, id.String())
		return
	}

	inRows, err := Db.Query(ctx, `
		SELECT txid, vout, status, amt
		FROM bsv20_txos
		WHERE spend=$1 AND id=$2`,
		txid,
		id,
	)
	if err != nil {
		log.Panicf("%x - %v\n", txid, err)
	}
	defer inRows.Close()

	var reason string
	var tokensIn uint64
	var tokenOuts []uint32
	for inRows.Next() {
		var inTxid []byte
		var vout uint32
		var amt uint64
		var inStatus int
		err = inRows.Scan(&inTxid, &vout, &inStatus, &amt)
		if err != nil {
			log.Panicf("%x - %v\n", txid, err)
		}

		switch inStatus {
		case -1:
			reason = "invalid input"
		case 0:
			fmt.Printf("inputs pending %s %x\n", id.String(), txid)
			return
		case 1:
			tokensIn += amt
		}
	}
	inRows.Close()

	// fmt.Println("TokensIn:", tokensIn)
	outRows, err := Db.Query(ctx, `
		SELECT vout, status, amt
		FROM bsv20_txos
		WHERE txid=$1 AND id=$2`,
		txid,
		id,
	)
	if err != nil {
		log.Panicf("%x - %v\n", txid, err)
	}
	defer outRows.Close()

	for outRows.Next() {
		var vout uint32
		var amt uint64
		var status int
		err = outRows.Scan(&vout, &status, &amt)
		if err != nil {
			log.Panicf("%x - %v\n", txid, err)
		}
		tokenOuts = append(tokenOuts, vout)
		if reason != "" {
			fmt.Println("Failed:", reason)
			continue
		}
		if amt > tokensIn {
			reason = fmt.Sprintf("insufficient balance %d < %d", tokensIn, amt)
			if !mined {
				fmt.Printf("%s %s - %x\n", id.String(), reason, txid)
				return
			}
		} else {
			tokensIn -= amt
		}
	}

	// fmt.Println("TokensOut:", len(tokenOuts))
	t, err := Db.Begin(ctx)
	if err != nil {
		log.Panic(err)
	}
	defer t.Rollback(ctx)

	if reason != "" {
		log.Printf("Transfer Invalid: %x %s %s\n", txid, id.String(), reason)
		_, err := t.Exec(ctx, `
				UPDATE bsv20_txos
				SET status=-1, reason=$3
				WHERE txid=$1 AND vout=ANY($2)`,
			txid,
			tokenOuts,
			reason,
		)
		if err != nil {
			log.Panicf("%x %v\n", txid, err)
		}
	} else {
		log.Printf("Transfer Valid: %x %s\n", txid, id.String())
		_, err := t.Exec(ctx, `
				UPDATE bsv20_txos
				SET status=1
				WHERE txid=$1 AND vout=ANY ($2)`,
			txid,
			tokenOuts,
		)
		if err != nil {
			log.Panicf("%x %v\n", txid, err)
		}
		if id != nil {
			_, err := t.Exec(ctx, `
					UPDATE bsv20_v2
					SET fund_used=fund_used+$2
					WHERE id=$1`,
				id,
				BSV20V2_OP_COST*len(tokenOuts),
			)
			if err != nil {
				log.Panicf("%x %v\n", txid, err)
			}
		}
	}

	err = t.Commit(ctx)
	if err != nil {
		log.Panic(err)
	}
	return len(tokenOuts)
}

func LoadTicker(tick string) (ticker *Bsv20) {
	rows, err := Db.Query(ctx, `
		SELECT txid, vout, height, idx, tick, max, lim, supply, fund_balance
		FROM bsv20
		WHERE tick=$1 AND status=1`,
		tick,
	)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	if rows.Next() {
		ticker = &Bsv20{}
		err = rows.Scan(&ticker.Txid, &ticker.Vout, &ticker.Height, &ticker.Idx, &ticker.Ticker, &ticker.Max, &ticker.Limit, &ticker.Supply, &ticker.FundBalance)
		if err != nil {
			log.Panicln(tick, err)
		}
	}
	return
}

func LoadTokenById(id *lib.Outpoint) (token *Bsv20) {
	rows, err := Db.Query(ctx, `
		SELECT id, height, idx, sym, icon, dec, amt, fund_balance
		FROM bsv20_v2
		WHERE id=$1`,
		id,
	)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	if rows.Next() {
		token = &Bsv20{}
		err = rows.Scan(&token.Id, &token.Height, &token.Idx, &token.Symbol, &token.Icon, &token.Decimals, &token.Amt, &token.FundBalance)
		if err != nil {
			log.Panicln(id, err)
		}
	}
	return
}

type V1TokenFunds struct {
	Tick   string
	Total  int64
	PKHash []byte
	Used   int64
}

func (t *V1TokenFunds) Save() {
	_, err := Db.Exec(ctx, `
		UPDATE bsv20
		SET fund_total=$2, fund_used=$3
		WHERE tick=$1`,
		t.Tick,
		t.Total,
		t.Used,
	)
	if err != nil {
		log.Panicln("SAVE", t.Tick, t.Total, t.Used, err)
		panic(err)
	}
	Rdb.Set(ctx, "funds:"+t.Tick, t.Balance(), 0)
}

func (t *V1TokenFunds) Balance() int64 {
	return t.Total - t.Used
}

func (t *V1TokenFunds) UpdateFunding() {
	var total sql.NullInt64
	row := Db.QueryRow(ctx, `
		SELECT SUM(satoshis)
		FROM bsv20 b
		JOIN txos t ON t.pkhash=b.fund_pkhash
		WHERE b.status=1 AND b.fund_pkhash=$1`,
		t.PKHash,
	)

	err := row.Scan(&total)
	if err != nil && err != pgx.ErrNoRows {
		log.Panicln(err)
	}
	t.Total = total.Int64

	row = Db.QueryRow(ctx, `
		SELECT COUNT(1) * $2
		FROM bsv20_txos
		WHERE tick=$1 AND status IN (-1, 1)`,
		t.Tick,
		BSV20V1_OP_COST,
	)
	err = row.Scan(&t.Used)
	if err != nil && err != pgx.ErrNoRows {
		log.Panicln(err)
	}

	t.Save()
}

type V2TokenFunds struct {
	Id     *lib.Outpoint
	Total  int64
	PKHash []byte
	Used   int64
}

func (t *V2TokenFunds) Save() {
	_, err := Db.Exec(ctx, `
		UPDATE bsv20_v2
		SET fund_total=$2, fund_used=$3
		WHERE id=$1`,
		t.Id,
		t.Total,
		t.Used,
	)
	if err != nil {
		panic(err)
	}
	Rdb.Set(ctx, "funds:"+t.Id.String(), t.Balance(), 0)
}

func (t *V2TokenFunds) Balance() int64 {
	return t.Total - t.Used
}

func (t *V2TokenFunds) UpdateFunding() {
	var total sql.NullInt64
	row := Db.QueryRow(ctx, `
		SELECT SUM(satoshis)
		FROM bsv20_v2 b
		JOIN txos t ON t.pkhash=b.fund_pkhash
		WHERE b.fund_pkhash=$1`,
		t.PKHash,
	)

	err := row.Scan(&total)
	if err != nil && err != pgx.ErrNoRows {
		log.Panicln(err)
	}
	t.Total = total.Int64

	row = Db.QueryRow(ctx, `
		SELECT COUNT(1) * $2
		FROM bsv20_txos
		WHERE id=$1 AND status IN (-1, 1)`,
		t.Id,
		BSV20V2_OP_COST,
	)
	err = row.Scan(&t.Used)
	if err != nil && err != pgx.ErrNoRows {
		log.Panicln(err)
	}

	t.Save()
}

func InitializeV1Funding(concurrency int) map[string]*V1TokenFunds {
	tickFunds := map[string]*V1TokenFunds{}
	limiter := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var m sync.Mutex
	rows, err := Db.Query(context.Background(), `
		SELECT DISTINCT tick, fund_pkhash
		FROM bsv20
		WHERE status=1`,
	)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	fmt.Println("Processing v1 funding")
	for rows.Next() {
		funds := &V1TokenFunds{}
		err = rows.Scan(&funds.Tick, &funds.PKHash)
		if err != nil {
			panic(err)
		}
		limiter <- struct{}{}
		wg.Add(1)
		go func(funds *V1TokenFunds) {
			defer func() {
				wg.Done()
				<-limiter
			}()
			add, err := bscript.NewAddressFromPublicKeyHash(funds.PKHash, true)
			if err != nil {
				log.Panicln(err)
			}
			url := fmt.Sprintf("%s/ord/%s", os.Getenv("INDEXER"), add.AddressString)
			// log.Println("URL:", url)
			resp, err := http.Get(url)
			if err != nil {
				log.Panicln(err)
			}
			defer resp.Body.Close()

			funds.UpdateFunding()
			m.Lock()
			tickFunds[funds.Tick] = funds
			m.Unlock()
		}(funds)
	}
	wg.Wait()
	return tickFunds
}

func InitializeV2Funding(concurrency int) map[string]*V2TokenFunds {
	idFunds := map[string]*V2TokenFunds{}
	limiter := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var m sync.Mutex
	rows, err := Db.Query(context.Background(), `
		SELECT id, fund_pkhash
		FROM bsv20_v2`,
	)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	fmt.Println("Processing v2 funding")
	for rows.Next() {
		funds := &V2TokenFunds{}
		err = rows.Scan(&funds.Id, &funds.PKHash)
		if err != nil {
			panic(err)
		}
		limiter <- struct{}{}
		wg.Add(1)
		go func(funds *V2TokenFunds) {
			defer func() {
				wg.Done()
				<-limiter
			}()
			add, err := bscript.NewAddressFromPublicKeyHash(funds.PKHash, true)
			if err != nil {
				log.Panicln(err)
			}
			url := fmt.Sprintf("%s/ord/%s", os.Getenv("INDEXER"), add.AddressString)
			log.Println("URL:", url)
			resp, err := http.Get(url)
			if err != nil {
				log.Panicln(err)
			}
			defer resp.Body.Close()

			funds.UpdateFunding()
			m.Lock()
			idFunds[funds.Id.String()] = funds
			m.Unlock()
		}(funds)
	}
	wg.Wait()
	return idFunds
}

func UpdateBsv20V2TokenFunding(funds *V2TokenFunds) {
	row := Db.QueryRow(ctx, `
		SELECT SUM(satoshis)
		FROM txos
		WHERE pkhash=$1`,
		funds.PKHash,
	)

	var total sql.NullInt64
	err := row.Scan(&total)
	if err != nil {
		log.Panicln(err)
	}
	funds.Total = total.Int64

	row = Db.QueryRow(ctx, `
		SELECT COUNT(1)
		FROM bsv20_txos
		WHERE id = $1 AND status IN (-1, 1)`,
		funds.Id,
	)
	var ops int
	err = row.Scan(&ops)
	if err != nil {
		log.Panicln(err)
	}
	funds.Used = int64(ops * BSV20V2_OP_COST)

	funds.Save()
}