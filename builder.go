package iso8583

import "sync"

// Builder pool for reuse
var builderPool = sync.Pool{
	New: func() interface{} {
		return &Builder{
			errors: make([]error, 0, 4),
		}
	},
}

type Builder struct {
	msg    *Message
	errors []error
}

func NewBuilder(opts ...MessageOption) *Builder {
	b := builderPool.Get().(*Builder)
	b.msg = NewMessage(opts...)
	b.errors = b.errors[:0]
	return b
}

// Release returns the builder to the pool
func (b *Builder) Release() {
	if b.msg != nil {
		b.msg = nil
	}
	b.errors = b.errors[:0]
	builderPool.Put(b)
}

func (b *Builder) MTI(mti string) *Builder {
	if len(mti) != 4 {
		b.errors = append(b.errors, ErrInvalidMTI)
		return b
	}
	copy(b.msg.mti[:], mti)
	return b
}

func (b *Builder) Field(fieldNum int, value interface{}) *Builder {
	if err := b.msg.SetField(fieldNum, value); err != nil {
		b.errors = append(b.errors, err)
	}
	return b
}

func (b *Builder) PAN(pan string) *Builder {
	return b.Field(2, pan)
}

func (b *Builder) ProcessingCode(code string) *Builder {
	return b.Field(3, code)
}

func (b *Builder) Amount(amount string) *Builder {
	return b.Field(4, amount)
}

func (b *Builder) STAN(stan string) *Builder {
	return b.Field(11, stan)
}

func (b *Builder) Build() (*Message, error) {
	if len(b.errors) > 0 {
		return nil, b.errors[0]
	}
	msg := b.msg
	b.msg = nil // Transfer ownership
	return msg, nil
}

func (b *Builder) MustBuild() *Message {
	if len(b.errors) > 0 {
		panic(b.errors[0])
	}
	msg := b.msg
	b.msg = nil // Transfer ownership
	return msg
}
