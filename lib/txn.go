package lib

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"

	"github.com/libsv/go-bt/v2"
)

type Status uint

var (
	Unconfirmed Status = 0
	Confirmed   Status = 1
)

type IndexResult struct {
	Txos          []*Txo          `json:"txos"`
	ParsedScripts []*ParsedScript `json:"parsed"`
	Spends        []*Txo          `json:"spends"`
}

func IndexSpends(tx *bt.Tx, save bool) (spends []*Txo, err error) {
	txid := tx.TxIDBytes()
	for vin, txin := range tx.Inputs {
		spend := &Txo{
			Txid:  txin.PreviousTxID(),
			Vout:  txin.PreviousTxOutIndex,
			Spend: txid,
			Vin:   uint32(vin),
		}
		spends = append(spends, spend)
		if save {
			err = spend.SaveSpend()
			if err != nil {
				return
			}
			var msg []byte
			msg, err = json.Marshal(spend)
			if err != nil {
				return
			}
			Rdb.Publish(context.Background(), hex.EncodeToString(spend.Lock), msg)
			if spend.Listing {
				Rdb.Publish(context.Background(), "unlist", msg)
			}
		}
	}
	return
}

func IndexTxos(tx *bt.Tx, height uint32, idx uint32, save bool) (result *IndexResult, err error) {
	txid := tx.TxIDBytes()
	result = &IndexResult{
		Txos:          []*Txo{},
		ParsedScripts: []*ParsedScript{},
	}
	var accSats uint64
	for vout, txout := range tx.Outputs {
		accSats += txout.Satoshis

		txo := &Txo{
			Txid:     txid,
			Vout:     uint32(vout),
			Height:   height,
			Idx:      idx,
			Satoshis: txout.Satoshis,
			AccSats:  accSats,
		}
		txo.Origin, err = LoadOrigin(txo)
		if err != nil {
			return
		}

		parsed := ParseScript(*txout.LockingScript, true)
		parsed.Txid = txid
		parsed.Vout = uint32(vout)
		parsed.Height = height
		parsed.Idx = idx
		parsed.Origin = txo.Origin
		if save {
			err = parsed.Save()
			if err != nil {
				return
			}
		}

		if save && len(parsed.Listings) > 0 {
			row := SetSpend.QueryRow(txid, uint32(vout))
			var origin Outpoint
			var lock []byte
			err = row.Scan(&lock, &origin)
			if err != nil {
				return
			}
			for seq, listing := range parsed.Listings {
				listing.ListTxid = txid
				listing.ListVout = uint32(vout)
				listing.ListSeq = uint32(seq)
				listing.Origin = origin
				err = listing.Save()
				if err != nil {
					return
				}
				var msg []byte
				msg, err = json.Marshal(listing)
				if err != nil {
					return
				}
				Rdb.Publish(context.Background(), hex.EncodeToString(lock), msg)
				Rdb.Publish(context.Background(), "list", msg)
			}
		}
		result.ParsedScripts = append(result.ParsedScripts, parsed)
		if txout.Satoshis != 1 {
			continue
		}

		txo.Lock = parsed.Lock

		result.Txos = append(result.Txos, txo)
		if save {
			err = txo.Save()
			if err != nil {
				return
			}
			var msg []byte
			msg, err = json.Marshal(txo)
			if err != nil {
				return
			}
			Rdb.Publish(context.Background(), hex.EncodeToString(txo.Lock), msg)
		}
	}
	return
}

func LoadOrigin(txo *Txo) (origin *Outpoint, err error) {
	if txo.Satoshis != 1 {
		return
	}
	rows, err := GetInput.Query(txo.Txid, txo.AccSats)
	if err != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		inTxo := &Txo{}
		err = rows.Scan(
			&inTxo.Txid,
			&inTxo.Vout,
			&inTxo.Satoshis,
			&inTxo.AccSats,
			&inTxo.Lock,
			&inTxo.Spend,
			&inTxo.Origin,
		)
		if err != nil {
			return
		}
		if inTxo.Origin != nil {
			origin = inTxo.Origin
			return
		}
	} else {
		o := Outpoint(binary.BigEndian.AppendUint32(txo.Txid, txo.Vout))
		origin = &o
	}

	return origin, nil
}
