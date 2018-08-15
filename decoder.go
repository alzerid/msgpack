package msgpack
import (
    "reflect"
    "unsafe"
    "bytes"
    "io"
)

type Unmarshaler interface {
    MsgPackUnmarshaler([]byte) error
}

type Token interface{}

type fixInt int8
type fixUint uint8

/**************************/
/** Start Misc Functions **/
/**************************/

// Function reads one byte of data
func readByte(rdr io.Reader) (byte, error) {
    if brdr, ok := rdr.(io.ByteReader); ok {
        return brdr.ReadByte()
    }

    buf := make([]byte, 1)
    if _, err := rdr.Read(buf); err != nil {
        return buf[0], err
    }

    return buf[0], nil
}

/************************/
/** End Misc Functions **/
/************************/

/*******************/
/** Start Decoder **/
/*******************/
type Decoder struct {
    rdr io.Reader
    eof bool
    k Kind
}

// Function creates a new decoder
func NewDecoder(r io.Reader) *Decoder {
    return &Decoder{ rdr: r }
}

// Method walks the reader and returns the token. Token
// can be primitive values, start/end of map, start/end
// of array, 
func (d *Decoder) Token() (Token, error) {
    // Read for control byte
    cbyte, err := readByte(d.rdr)
    if err != nil {
        return nil, err
    }

    //Determine what to do with the control byte
    //kbyte := Kind(cbyte)
    switch Kind(cbyte) {

        //Unsigned integers
        case Uint64:
            var buf [8]byte
            if _, err := d.rdr.Read(buf[:]); err != nil {
                return nil, err
            }

            d.k = Uint64
            reverseByte(buf[:])
            ret := (*uint64)(unsafe.Pointer(&buf))
            return Token(*ret), nil

        case Uint32:
            var buf [4]byte
            if _, err := d.rdr.Read(buf[:]); err != nil {
                return nil, err
            }

            d.k = Uint32
            reverseByte(buf[:])
            ret := (*uint32)(unsafe.Pointer(&buf))
            return Token(*ret), nil

        case Uint16:
            var buf [2]byte
            if _, err := d.rdr.Read(buf[:]); err != nil {
                return nil, err
            }

            d.k = Uint16
            reverseByte(buf[:])
            ret := (*uint16)(unsafe.Pointer(&buf))
            return Token(*ret), nil

        case Uint8:
            var buf [1]byte
            if _, err := d.rdr.Read(buf[:]); err != nil {
                return nil, err
            }

            d.k = Uint8
            reverseByte(buf[:])
            ret := (*uint8)(unsafe.Pointer(&buf))
            return Token(*ret), nil

        //Signed
        case Int64:
            var buf [8]byte
            if _, err := d.rdr.Read(buf[:]); err != nil {
                return nil, err
            }

            d.k = Int64
            reverseByte(buf[:])
            ret := (*int64)(unsafe.Pointer(&buf))
            return Token(*ret), nil

        case Int32:
            var buf [4]byte
            if _, err := d.rdr.Read(buf[:]); err != nil {
                return nil, err
            }

            d.k = Int32
            reverseByte(buf[:])
            ret := (*int32)(unsafe.Pointer(&buf))
            return Token(*ret), nil

        case Int16:
            var buf [2]byte
            if _, err := d.rdr.Read(buf[:]); err != nil {
                return nil, err
            }

            d.k = Int16
            reverseByte(buf[:])
            ret := (*int16)(unsafe.Pointer(&buf))
            return Token(*ret), nil

        case Int8:
            var buf [1]byte
            if _, err := d.rdr.Read(buf[:]); err != nil {
                return nil, err
            }

            d.k = Int8
            ret := (*int8)(unsafe.Pointer(&buf))
            return Token(*ret), nil
    }

    //Fix num
    if Kind(cbyte) & 0x80 == FixUint {
        d.k = FixUint
        ret := (*uint8)(unsafe.Pointer(&cbyte))
        return Token(*ret), nil
    } else if Kind(cbyte) & FixInt == FixInt {
        d.k = FixInt

        //Ones compliment the number
        ret := *(*uint8)(unsafe.Pointer(&cbyte)) & (^(uint8(FixInt)))
        ret = (ret ^ 0xff) + 1
        return int8(ret), nil
    }

    return nil, nil
}

// Method decodes the item using the reflect types
func (d *Decoder) decode(rv reflect.Value) error {

    //Got pointer so deref it
    kind := rv.Kind()
    if kind == reflect.Ptr {
        return d.decode(rv.Elem())
    }

    //Get token
    tok, err := d.Token()
    if err != nil {
        return err
    }

    //Switch based on token
    switch v := tok.(type) {

        //Signed Integer
        case int64:
            rv.SetInt(v)
        case int32:
            rv.SetInt(int64(v))
        case int16:
            rv.SetInt(int64(v))
        case int8:
            rv.SetInt(int64(v))
        case int:
            rv.SetInt(int64(v))

        //Unsigned Integer
        case uint64:
            rv.SetUint(v)
        case uint32:
            d.k = Uint32
            rv.SetUint(uint64(v))
        case uint16:
            rv.SetUint(uint64(v))
        case uint8:
            rv.SetUint(uint64(v))
        case uint:
            rv.SetUint(uint64(v))
    }

    return nil
}

// Method decodes an interface
func (d *Decoder) Decode(v interface{}) error {
    return d.decode(reflect.ValueOf(v))
}

// Gets current kind
func (d *Decoder) Kind() Kind {
    return d.k
}

/*****************/
/** End Decoder **/
/*****************/

// Function Unmarshals the data
func Unmarshal(d []byte, v interface{}) error {
    dec := NewDecoder(bytes.NewReader(d))
    if err := dec.Decode(v); err != nil {
        return err
    }

    return nil
}
