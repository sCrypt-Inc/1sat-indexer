package lib

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bitcoinschema/go-bitcoin"
	"github.com/libsv/go-bt/v2"
	"github.com/libsv/go-bt/v2/bscript"
)

var PATTERN []byte
var MAP = "1PuQa7K62MiKCtssSLKy1kh56WWU7MtUR5"
var B = "19HxigV4QyBv3tHpQVcUEQyq1pzZVdoAut"

// var REG = "1regEua3EpFKCSuYGgB77rbEUDFNGJP3P"

var OrdLockPrefix []byte
var OrdLockSuffix []byte
var OpNSPrefix []byte
var OpNSSuffix []byte

var selfRef *regexp.Regexp

func init() {
	val, err := hex.DecodeString("0063036f7264")
	if err != nil {
		log.Panic(err)
	}
	PATTERN = val

	OrdLockPrefix, _ = hex.DecodeString("2097dfd76851bf465e8f715593b217714858bbe9570ff3bd5e33840a34e20ff0262102ba79df5f8ae7604a9830f03c7933028186aede0675a16f025dc4f8be8eec0382201008ce7480da41702918d1ec8e6849ba32b4d65b1e40dc669c31a1e6306b266c0000")
	OrdLockSuffix, _ = hex.DecodeString("615179547a75537a537a537a0079537a75527a527a7575615579008763567901c161517957795779210ac407f0e4bd44bfc207355a778b046225a7068fc59ee7eda43ad905aadbffc800206c266b30e6a1319c66dc401e5bd6b432ba49688eecd118297041da8074ce081059795679615679aa0079610079517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e01007e81517a75615779567956795679567961537956795479577995939521414136d08c5ed2bf3ba048afe6dcaebafeffffffffffffffffffffffffffffff00517951796151795179970079009f63007952799367007968517a75517a75517a7561527a75517a517951795296a0630079527994527a75517a6853798277527982775379012080517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e01205279947f7754537993527993013051797e527e54797e58797e527e53797e52797e57797e0079517a75517a75517a75517a75517a75517a75517a75517a75517a75517a75517a75517a75517a756100795779ac517a75517a75517a75517a75517a75517a75517a75517a75517a7561517a75517a756169587951797e58797eaa577961007982775179517958947f7551790128947f77517a75517a75618777777777777777777767557951876351795779a9876957795779ac777777777777777767006868")
	OpNSPrefix, _ = hex.DecodeString("0168016a2097dfd76851bf465e8f715593b217714858bbe9570ff3bd5e33840a34e20ff0262102ba79df5f8ae7604a9830f03c7933028186aede0675a16f025dc4f8be8eec0382201008ce7480da41702918d1ec8e6849ba32b4d65b1e40dc669c31a1e6306b266c00000000000000")
	OpNSSuffix, _ = hex.DecodeString("615179597a75587a587a587a587a587a587a587a587a0079587a75577a577a577a577a577a577a577a00577a75567a567a567a567a567a567a00567a75557a557a557a557a557a00557a75547a547a547a547a00547a75537a537a537a7575615c7961007901687f776100005279517f75007f77007901fd87635379537f75517f7761007901007e81517a7561537a75527a527a5379535479937f75537f77527a75517a67007901fe87635379557f75517f7761007901007e81517a7561537a75527a527a5379555479937f75557f77527a75517a67007901ff87635379597f75517f7761007901007e81517a7561537a75527a527a5379595479937f75597f77527a75517a675379517f75007f7761007901007e81517a7561537a75527a527a5379515479937f75517f77527a75517a6868685179517a75517a75517a75517a7561517a7561007961007982775179517951947f755179549451947f77007981527951799454945194517a75517a75517a75517a7561517951797f75537a75527a527a0000537953797f77610079537a75527a527a00527a75517a7561615179517951937f7551797f775179768b537a75527a527a75010051798791517a75610079916361005379005179557951937f7555797f77815579768b577a75567a567a567a567a567a567a750079014c9f630079547a75537a537a537a527956795579937f7556797f77527a75517a670079014c9c635279567951937f7556797f7761007901007e81517a7561547a75537a537a537a55795193567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670079014d9c635279567952937f7556797f7761007901007e81517a7561547a75537a537a537a55795293567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670079014e9c635279567954937f7556797f7761007901007e81517a7561547a75537a537a537a55795493567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670069686868685579547993567a75557a557a557a557a557a5579755179517a75517a75517a75517a75615a7a75597a597a597a597a597a597a597a597a597a6161005379005179557951937f7555797f77815579768b577a75567a567a567a567a567a567a750079014c9f630079547a75537a537a537a527956795579937f7556797f77527a75517a670079014c9c635279567951937f7556797f7761007901007e81517a7561547a75537a537a537a55795193567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670079014d9c635279567952937f7556797f7761007901007e81517a7561547a75537a537a537a55795293567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670079014e9c635279567954937f7556797f7761007901007e81517a7561547a75537a537a537a55795493567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670069686868685579547993567a75557a557a557a557a557a5579755179517a75517a75517a75517a75618161597a75587a587a587a587a587a587a587a587a61005379005179557951937f7555797f77815579768b577a75567a567a567a567a567a567a750079014c9f630079547a75537a537a537a527956795579937f7556797f77527a75517a670079014c9c635279567951937f7556797f7761007901007e81517a7561547a75537a537a537a55795193567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670079014d9c635279567952937f7556797f7761007901007e81517a7561547a75537a537a537a55795293567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670079014e9c635279567954937f7556797f7761007901007e81517a7561547a75537a537a537a55795493567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670069686868685579547993567a75557a557a557a557a557a5579755179517a75517a75517a75517a7561587a75577a577a577a577a577a577a577a61005379005179557951937f7555797f77815579768b577a75567a567a567a567a567a567a750079014c9f630079547a75537a537a537a527956795579937f7556797f77527a75517a670079014c9c635279567951937f7556797f7761007901007e81517a7561547a75537a537a537a55795193567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670079014d9c635279567952937f7556797f7761007901007e81517a7561547a75537a537a537a55795293567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670079014e9c635279567954937f7556797f7761007901007e81517a7561547a75537a537a537a55795493567a75557a557a557a557a557a557975527956795579937f7556797f77527a75517a670069686868685579547993567a75557a557a557a557a557a5579755179517a75517a75517a75517a7561577a75567a567a567a567a567a567a6801117901c1615179011179011179210ac407f0e4bd44bfc207355a778b046225a7068fc59ee7eda43ad905aadbffc800206c266b30e6a1319c66dc401e5bd6b432ba49688eecd118297041da8074ce08100113795679615679aa0079610079517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e01007e81517a75615779567956795679567961537956795479577995939521414136d08c5ed2bf3ba048afe6dcaebafeffffffffffffffffffffffffffffff00517951796151795179970079009f63007952799367007968517a75517a75517a7561527a75517a517951795296a0630079527994527a75517a6853798277527982775379012080517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f517f7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e7c7e01205279947f7754537993527993013051797e527e54797e58797e527e53797e52797e57797e0079517a75517a75517a75517a75517a75517a75517a75517a75517a75517a75517a75517a75517a756100795779ac517a75517a75517a75517a75517a75517a75517a75517a75517a7561517a75517a75616959798277009c6301117961007901687f7501447f77517a756101207f75007f7701127961007901687f7501447f77517a756101207f778154807e5a7a75597a597a597a597a597a597a597a597a597a6801157901157961587952797e51797eaa007961007901007e81517a75610200015f799461007900a269517951796100517958968002010052795897987e81517a7561965179009c63527967527952974f9c6300795194675279009f630079009c670068634f670079686868517a75517a75517a75610079009c695179517a75517a75517a75517a7561577a75567a567a567a567a567a567a011579610079012d9c6400790130a2630079013a9f6700686751686400790161a2630079017b9f67006867516800796951527961007900a269517951796100517958968002010052795897987e81517a756195517a75517a75610079825d79827ba4766b807c6c808481009c695b79517993517a75517a75517a7561597a75587a587a587a587a587a587a587a587a51616100610079635167010068517a75615b796100798277005179014c9f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a756751790200019f63014c527951615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179030000019f63014d527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f63014e527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7567006968686868007953797e517a75517a75517a75617e5a79610079009c630100670079686100798277005179014c9f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a756751790200019f63014c527951615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179030000019f63014d527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f63014e527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7567006968686868007953797e517a75517a75517a7561517a75617e59796100798277005179014c9f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a756751790200019f63014c527951615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179030000019f63014d527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f63014e527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7567006968686868007953797e517a75517a75517a75617e58796100798277005179014c9f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a756751790200019f63014c527951615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179030000019f63014d527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f63014e527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7567006968686868007953797e517a75517a75517a75617e5779517961007982775480517951797e0051807e517a75517a75617e517a75610079527961007958805279610079827700517902fd009f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a75675179030000019f6301fd527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f6301fe527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179090000000000000000019f6301ff527958615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7568686868007953797e517a75517a75517a75617e517a75517a7561517a75517a7561005a7a75597a597a597a597a597a597a597a597a597a58790117797e597a75587a587a587a587a587a587a587a587a51616100610079635167010068517a75615c796100798277005179014c9f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a756751790200019f63014c527951615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179030000019f63014d527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f63014e527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7567006968686868007953797e517a75517a75517a75617e5b79610079009c630100670079686100798277005179014c9f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a756751790200019f63014c527951615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179030000019f63014d527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f63014e527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7567006968686868007953797e517a75517a75517a7561517a75617e5a796100798277005179014c9f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a756751790200019f63014c527951615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179030000019f63014d527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f63014e527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7567006968686868007953797e517a75517a75517a75617e59796100798277005179014c9f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a756751790200019f63014c527951615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179030000019f63014d527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f63014e527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7567006968686868007953797e517a75517a75517a75617e5879517961007982775480517951797e0051807e517a75517a75617e517a75610079527961007958805279610079827700517902fd009f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a75675179030000019f6301fd527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f6301fe527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179090000000000000000019f6301ff527958615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7568686868007953797e517a75517a75517a75617e517a75517a7561517a75517a7561011579615a795f79827700a0635b79012e7e60797e517a756851791a0063036f726451116170706c69636174696f6e2f6f702d6e73007e517982777e51797e0115797e0114797e01217e21316f704e53554a56624263325666384c464e536f797747474b346a4d63475672437e01247e5e797e517a75517a75615161007958805279610079827700517902fd009f63517951615179517951938000795179827751947f75007f77517a75517a75517a7561517a75675179030000019f6301fd527952615179517951938000795179827751947f75007f77517a75517a75517a75617e517a756751790500000000019f6301fe527954615179517951938000795179827751947f75007f77517a75517a75517a75617e517a75675179090000000000000000019f6301ff527958615179517951938000795179827751947f75007f77517a75517a75517a75617e517a7568686868007953797e517a75517a75517a75617e517a75517a7561527952797e51797e0116797e0079aa01167961007982775179517958947f7551790128947f77517a75517a7561877777777777777777777777777777777777777777777777777777")
	selfRef = regexp.MustCompile(`^_\d+$`)
}

type OpPart struct {
	OpCode byte
	Data   []byte
	Len    uint32
}

func ReadOp(b []byte, idx *int) (op *OpPart, err error) {
	if len(b) <= *idx {
		// log.Panicf("ReadOp: %d %d", len(b), *idx)
		err = fmt.Errorf("ReadOp: %d %d", len(b), *idx)
		return
	}
	switch b[*idx] {
	case bscript.OpPUSHDATA1:
		if len(b) < *idx+2 {
			err = bscript.ErrDataTooSmall
			return
		}

		l := int(b[*idx+1])
		*idx += 2

		if len(b) < *idx+l {
			err = bscript.ErrDataTooSmall
			return
		}

		op = &OpPart{OpCode: bscript.OpPUSHDATA1, Data: b[*idx : *idx+l]}
		*idx += l

	case bscript.OpPUSHDATA2:
		if len(b) < *idx+3 {
			err = bscript.ErrDataTooSmall
			return
		}

		l := int(binary.LittleEndian.Uint16(b[*idx+1:]))
		*idx += 3

		if len(b) < *idx+l {
			err = bscript.ErrDataTooSmall
			return
		}

		op = &OpPart{OpCode: bscript.OpPUSHDATA2, Data: b[*idx : *idx+l]}
		*idx += l

	case bscript.OpPUSHDATA4:
		if len(b) < *idx+5 {
			err = bscript.ErrDataTooSmall
			return
		}

		l := int(binary.LittleEndian.Uint32(b[*idx+1:]))
		*idx += 5

		if len(b) < *idx+l {
			err = bscript.ErrDataTooSmall
			return
		}

		op = &OpPart{OpCode: bscript.OpPUSHDATA4, Data: b[*idx : *idx+l]}
		*idx += l

	default:
		if b[*idx] >= 0x01 && b[*idx] < bscript.OpPUSHDATA1 {
			l := b[*idx]
			if len(b) < *idx+int(1+l) {
				err = bscript.ErrDataTooSmall
				return
			}
			op = &OpPart{OpCode: b[*idx], Data: b[*idx+1 : *idx+int(l+1)]}
			*idx += int(1 + l)
		} else {
			op = &OpPart{OpCode: b[*idx]}
			*idx++
		}
	}

	return
}

func ParseBitcom(txo *Txo, idx *int) (err error) {
	script := *txo.Tx.Outputs[txo.Vout].LockingScript
	tx := txo.Tx
	d := txo.Data

	startIdx := *idx
	op, err := ReadOp(script, idx)
	if err != nil {
		return
	}
	switch string(op.Data) {
	case MAP:
		op, err = ReadOp(script, idx)
		if err != nil {
			return
		}
		if string(op.Data) != "SET" {
			return nil
		}
		d.Map = map[string]interface{}{}
		for {
			prevIdx := *idx
			op, err = ReadOp(script, idx)
			if err != nil || op.OpCode == bscript.OpRETURN || (op.OpCode == 1 && op.Data[0] == '|') {
				*idx = prevIdx
				break
			}
			opKey := op.Data
			prevIdx = *idx
			op, err = ReadOp(script, idx)
			if err != nil || op.OpCode == bscript.OpRETURN || (op.OpCode == 1 && op.Data[0] == '|') {
				*idx = prevIdx
				break
			}

			if len(opKey) > 256 || len(op.Data) > 1024 {
				continue
			}

			if !utf8.Valid(opKey) || !utf8.Valid(op.Data) {
				continue
			}

			if len(opKey) == 1 && opKey[0] == 0 {
				opKey = []byte{}
			}
			if len(op.Data) == 1 && op.Data[0] == 0 {
				op.Data = []byte{}
			}

			d.Map[string(opKey)] = string(op.Data)

		}
		if val, ok := d.Map["subTypeData"]; ok {
			var subTypeData json.RawMessage
			if err := json.Unmarshal([]byte(val.(string)), &subTypeData); err == nil {
				d.Map["subTypeData"] = subTypeData
			}
		}
		return nil
	case B:
		d.B = &File{}
		for i := 0; i < 4; i++ {
			prevIdx := *idx
			op, err = ReadOp(script, idx)
			if err != nil || op.OpCode == bscript.OpRETURN || (op.OpCode == 1 && op.Data[0] == '|') {
				*idx = prevIdx
				break
			}

			switch i {
			case 0:
				d.B.Content = op.Data
			case 1:
				d.B.Type = string(op.Data)
			case 2:
				d.B.Encoding = string(op.Data)
			case 3:
				d.B.Name = string(op.Data)
			}
		}
		hash := sha256.Sum256(d.B.Content)
		d.B.Size = uint32(len(d.B.Content))
		d.B.Hash = hash[:]
	case "SIGMA":
		sigma := &Sigma{}
		for i := 0; i < 4; i++ {
			prevIdx := *idx
			op, err = ReadOp(script, idx)
			if err != nil || op.OpCode == bscript.OpRETURN || (op.OpCode == 1 && op.Data[0] == '|') {
				*idx = prevIdx
				break
			}

			switch i {
			case 0:
				sigma.Algorithm = string(op.Data)
			case 1:
				sigma.Address = string(op.Data)
			case 2:
				sigma.Signature = op.Data
			case 3:
				vin, err := strconv.ParseInt(string(op.Data), 10, 32)
				if err == nil {
					sigma.Vin = uint32(vin)
				}
			}
		}
		d.Sigmas = append(d.Sigmas, sigma)

		outpoint := tx.Inputs[sigma.Vin].PreviousTxID()
		outpoint = binary.LittleEndian.AppendUint32(outpoint, tx.Inputs[sigma.Vin].PreviousTxOutIndex)
		// fmt.Printf("outpoint %x\n", outpoint)
		inputHash := sha256.Sum256(outpoint)
		// fmt.Printf("ihash: %x\n", inputHash)
		var scriptBuf []byte
		if script[startIdx-1] == bscript.OpRETURN {
			scriptBuf = script[:startIdx-1]
		} else if script[startIdx-1] == '|' {
			scriptBuf = script[:startIdx-2]
		} else {
			return nil
		}
		// fmt.Printf("scriptBuf %x\n", scriptBuf)
		outputHash := sha256.Sum256(scriptBuf)
		// fmt.Printf("ohash: %x\n", outputHash)
		msgHash := sha256.Sum256(append(inputHash[:], outputHash[:]...))
		// fmt.Printf("msghash: %x\n", msgHash)
		err = bitcoin.VerifyMessage(sigma.Address,
			base64.StdEncoding.EncodeToString(sigma.Signature),
			string(msgHash[:]),
		)
		if err != nil {
			fmt.Println("Error verifying signature", err)
			return nil
		}
		sigma.Valid = true
	default:
		*idx--
	}
	return
}

func ParseScript(txo *Txo) {
	d := &TxoData{}
	txo.Data = d
	script := *txo.Tx.Outputs[txo.Vout].LockingScript

	start := 0
	if len(script) >= 25 && bscript.NewFromBytes(script[:25]).IsP2PKH() {
		txo.PKHash = []byte(script[3:23])
		start = 25
	}

	var opFalse int
	var opIf int
	var opReturn int
	for i := start; i < len(script); {
		startI := i
		op, err := ReadOp(script, &i)
		if err != nil {
			break
		}
		// fmt.Println(prevI, i, op)
		switch op.OpCode {
		case bscript.Op0:
			opFalse = startI
		case bscript.OpIF:
			opIf = startI
		case bscript.OpRETURN:
			if opReturn == 0 {
				opReturn = startI
			}
			err = ParseBitcom(txo, &i)
			if err != nil {
				log.Println("Error parsing bitcom", err)
				continue
			}
		case bscript.OpDATA1:
			if op.Data[0] == '|' && opReturn > 0 {
				err = ParseBitcom(txo, &i)
				if err != nil {
					log.Println("Error parsing bitcom", err)
					continue
				}
			}
		}

		if bytes.Equal(op.Data, []byte("ord")) && opIf == startI-1 && opFalse == startI-2 {
			ins := &Inscription{
				File: &File{},
			}
		ordLoop:
			for {
				op, err = ReadOp(script, &i)
				if err != nil {
					break
				}
				switch op.OpCode {
				case bscript.Op0:
					op, err = ReadOp(script, &i)
					if err != nil {
						break ordLoop
					}
					ins.File.Content = op.Data
				case bscript.Op1:
					// case bscript.OpDATA1:
					// 	if op.OpCode == bscript.OpDATA1 && op.Data[0] != 1 {
					// 		continue
					// 	}
					op, err = ReadOp(script, &i)
					if err != nil {
						break ordLoop
					}
					if utf8.Valid(op.Data) {
						if len(op.Data) <= 256 {
							ins.File.Type = string(op.Data)
						} else {
							ins.File.Type = string(op.Data[:256])
						}
					}
				case bscript.OpENDIF:
					break ordLoop
				}
			}
			ins.File.Size = uint32(len(ins.File.Content))
			hash := sha256.Sum256(ins.File.Content)
			ins.File.Hash = hash[:]
			d.Inscription = ins
			insType := "file"
			if ins.File.Size <= 1024 && utf8.Valid(ins.File.Content) && !bytes.Contains(ins.File.Content, []byte{0}) {
				mime := strings.ToLower(ins.File.Type)
				if strings.HasPrefix(mime, "application/bsv-20") ||
					strings.HasPrefix(mime, "text/plain") ||
					strings.HasPrefix(mime, "application/json") {

					var data json.RawMessage
					err = json.Unmarshal(ins.File.Content, &data)
					if err == nil {
						insType = "json"
						ins.Json = data
						if strings.HasPrefix(mime, "application/bsv-20") {
							d.Bsv20, _ = parseBsv20(ins.File, txo.Height)
						}
						if txo.Height != nil && *txo.Height < 793000 &&
							strings.HasPrefix(mime, "text/plain") {
							d.Bsv20, _ = parseBsv20(ins.File, txo.Height)
						}
						if d.Bsv20 != nil {
							txo.Data.Types = append(txo.Data.Types, "bsv20")
						}
					}
				}
				if strings.HasPrefix(mime, "text") {
					if insType == "file" {
						insType = "text"
					}
					ins.Text = string(ins.File.Content)
					re := regexp.MustCompile(`\W`)

					words := map[string]struct{}{}
					for _, word := range re.Split(ins.Text, -1) {
						if len(word) > 1 {
							words[word] = struct{}{}
						}
					}

					if len(words) > 1 {
						ins.Words = make([]string, 0, len(words))
						for word := range words {
							ins.Words = append(ins.Words, word)
						}
					}
				}
			}
			if ins.File.Type == "application/op-reg" {
				err = json.Unmarshal(ins.File.Content, &txo.Data.Claims)
				if err == nil {
					txo.Data.Types = append(txo.Data.Types, "op-reg")
				}
			}
			txo.Data.Types = append(txo.Data.Types, insType)
		}
	}

	ordLockPrefixIndex := bytes.Index(script, OrdLockPrefix)
	ordLockSuffixIndex := bytes.Index(script, OrdLockSuffix)
	if ordLockPrefixIndex > -1 && ordLockSuffixIndex > len(OrdLockPrefix) {
		ordLock := script[ordLockPrefixIndex+len(OrdLockPrefix) : ordLockSuffixIndex]
		if ordLockParts, err := bscript.DecodeParts(ordLock); err == nil {
			txo.PKHash = ordLockParts[0]
			payOutput := &bt.Output{}
			_, err = payOutput.ReadFrom(bytes.NewReader(ordLockParts[1]))
			if err == nil {
				d.Listing = &Listing{
					Price:  payOutput.Satoshis,
					PayOut: payOutput.Bytes(),
				}
			}
		}
	}

	// opNSPrefixIndex := bytes.Index(script, OpNSPrefix)
	// opNSSuffixIndex := bytes.Index(script, OpNSSuffix)
	// if opNSPrefixIndex > -1 && opNSSuffixIndex > len(OpNSPrefix) {
	// }
}
