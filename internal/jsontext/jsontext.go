package jsontext

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

type Literal struct {
	raw string
}

func String(value string) Literal {
	return Literal{raw: encodeString(value)}
}

func Bool(value bool) Literal {
	if value {
		return Literal{raw: "true"}
	}
	return Literal{raw: "false"}
}

func encodeString(value string) string {
	var b []byte
	b = append(b, '"')
	for _, r := range value {
		switch r {
		case '\\':
			b = append(b, '\\', '\\')
		case '"':
			b = append(b, '\\', '"')
		case '\n':
			b = append(b, '\\', 'n')
		case '\r':
			b = append(b, '\\', 'r')
		case '\t':
			b = append(b, '\\', 't')
		default:
			if r < 0x20 {
				b = append(b, fmt.Sprintf("\\u%04x", r)...)
			} else {
				b = utf8.AppendRune(b, r)
			}
		}
	}
	b = append(b, '"')
	return string(b)
}

type Error struct {
	Detail string
}

func (e Error) Error() string {
	if e.Detail == "" {
		return "JSON не объект"
	}
	return e.Detail
}

func notObject() error { return Error{} }

func broken(detail string) error { return Error{Detail: detail} }

func Set(source, key string, value Literal) (string, error) {
	parser := parser{source: source}
	tree, err := parser.parseObject()
	if err != nil {
		return "", err
	}
	for _, match := range tree.properties {
		if match.key == key {
			return replace(source, match.valueStart, match.valueEnd, value.raw), nil
		}
	}
	insertion, err := insertionText(source, tree.closing, key, value.raw)
	if err != nil {
		return "", err
	}
	return replace(source, tree.closing, tree.closing, insertion), nil
}

func Remove(source, key string) (string, error) {
	parser := parser{source: source}
	tree, err := parser.parseObject()
	if err != nil {
		return "", err
	}
	var match *property
	for i := range tree.properties {
		if tree.properties[i].key == key {
			match = &tree.properties[i]
			break
		}
	}
	if match == nil {
		return source, nil
	}
	start, end := deletionSpan(source, *match, tree.properties)
	return replace(source, start, end, ""), nil
}

func insertionText(source string, closing int, key, encoded string) (string, error) {
	head := source[:closing]
	index := len(head)
	for index > 0 {
		r, size := utf8.DecodeLastRuneInString(head[:index])
		if unicode.IsSpace(r) {
			index -= size
			continue
		}
		break
	}
	needsComma := index > 0 && runeAt(head, index-1) != '{'
	comma := ""
	if needsComma {
		comma = ","
	}
	return comma + "\n  " + encodeString(key) + ": " + encoded + "\n", nil
}

func runeAt(s string, byteIndex int) rune {
	r, _ := utf8.DecodeLastRuneInString(s[:byteIndex+1])
	return r
}

func deletionSpan(source string, prop property, properties []property) (int, int) {
	othersAfter := false
	for _, item := range properties {
		if item.keyStart > prop.valueEnd {
			othersAfter = true
			break
		}
	}
	if othersAfter {
		end := prop.valueEnd
		index := end
		for index < len(source) {
			r, size := utf8.DecodeRuneInString(source[index:])
			if unicode.IsSpace(r) {
				index += size
				continue
			}
			break
		}
		if index < len(source) && source[index] == ',' {
			end = index + 1
		}
		return prop.keyStart, end
	}
	start := prop.keyStart
	index := start
	for index > 0 {
		r, size := utf8.DecodeLastRuneInString(source[:index])
		if unicode.IsSpace(r) {
			index -= size
			continue
		}
		if r == ',' {
			start = index - size
		}
		break
	}
	return start, prop.valueEnd
}

func replace(source string, start, end int, text string) string {
	return source[:start] + text + source[end:]
}

type property struct {
	key        string
	keyStart   int
	valueStart int
	valueEnd   int
}

type objectTree struct {
	properties []property
	closing    int
}

type parser struct {
	source string
	i      int
}

func (p *parser) parseObject() (objectTree, error) {
	p.skipTrivia()
	if !p.eat('{') {
		return objectTree{}, notObject()
	}
	var properties []property
	for {
		p.skipTrivia()
		if p.eat('}') {
			return objectTree{properties: properties, closing: p.i - 1}, nil
		}
		keyStart := p.i
		key, err := p.readString()
		if err != nil {
			return objectTree{}, err
		}
		p.skipTrivia()
		if !p.eat(':') {
			return objectTree{}, broken("нет двоеточия у " + key)
		}
		p.skipTrivia()
		valueStart := p.i
		if err := p.skipValue(); err != nil {
			return objectTree{}, err
		}
		properties = append(properties, property{
			key:        key,
			keyStart:   keyStart,
			valueStart: valueStart,
			valueEnd:   p.i,
		})
		p.skipTrivia()
		if p.eat(',') {
			continue
		}
		p.skipTrivia()
		if !p.eat('}') {
			return objectTree{}, broken("объект не закрыт")
		}
		return objectTree{properties: properties, closing: p.i - 1}, nil
	}
}

func (p *parser) skipTrivia() {
	for p.i < len(p.source) {
		r, size := utf8.DecodeRuneInString(p.source[p.i:])
		if unicode.IsSpace(r) {
			p.i += size
			continue
		}
		if r == '/' {
			next := p.i + size
			if next < len(p.source) && p.source[next] == '/' {
				p.i = next
				for p.i < len(p.source) && p.source[p.i] != '\n' {
					_, n := utf8.DecodeRuneInString(p.source[p.i:])
					p.i += n
				}
				continue
			}
			if next < len(p.source) && p.source[next] == '*' {
				p.i = next + 1
				for p.i < len(p.source) {
					if p.source[p.i] == '*' {
						after := p.i + 1
						if after < len(p.source) && p.source[after] == '/' {
							p.i = after + 1
							break
						}
					}
					_, n := utf8.DecodeRuneInString(p.source[p.i:])
					p.i += n
				}
				continue
			}
		}
		break
	}
}

func (p *parser) eat(expected rune) bool {
	if p.i >= len(p.source) {
		return false
	}
	r, size := utf8.DecodeRuneInString(p.source[p.i:])
	if r != expected {
		return false
	}
	p.i += size
	return true
}

func (p *parser) readString() (string, error) {
	if !p.eat('"') {
		return "", broken("ожидалась строка")
	}
	var value []rune
	for p.i < len(p.source) {
		r, size := utf8.DecodeRuneInString(p.source[p.i:])
		p.i += size
		if r == '\\' {
			if p.i >= len(p.source) {
				return "", broken("оборванный escape")
			}
			escaped, n := utf8.DecodeRuneInString(p.source[p.i:])
			p.i += n
			value = append(value, escaped)
			continue
		}
		if r == '"' {
			return string(value), nil
		}
		value = append(value, r)
	}
	return "", broken("строка не закрыта")
}

func (p *parser) skipValue() error {
	if p.i >= len(p.source) {
		return broken("нет значения")
	}
	r, _ := utf8.DecodeRuneInString(p.source[p.i:])
	switch r {
	case '"':
		_, err := p.readString()
		return err
	case '{':
		_, err := p.parseObject()
		return err
	case '[':
		return p.skipArray()
	}
	for p.i < len(p.source) {
		current, size := utf8.DecodeRuneInString(p.source[p.i:])
		if current == ',' || current == '}' || current == ']' || unicode.IsSpace(current) || current == '/' {
			break
		}
		p.i += size
	}
	return nil
}

func (p *parser) skipArray() error {
	if !p.eat('[') {
		return broken("ожидался массив")
	}
	for {
		p.skipTrivia()
		if p.eat(']') {
			return nil
		}
		if err := p.skipValue(); err != nil {
			return err
		}
		p.skipTrivia()
		if p.eat(',') {
			continue
		}
		p.skipTrivia()
		if !p.eat(']') {
			return broken("массив не закрыт")
		}
		return nil
	}
}
