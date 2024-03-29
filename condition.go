package condition

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/pkg/errors"
)

var (
	ErrParseData   = errors.New("parse data error")
	ErrNotFoundCol = errors.New("not field")
)

type Condition struct {
	tree IExprContext
}

func New(conditionExpr string) (*Condition, error) {
	conError := NewConditionError()
	// 新家一个 CharStream
	input := antlr.NewInputStream(conditionExpr)

	lexer := NewConditionLexer(input)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(conError)

	// 新建一个此法符号的缓冲区，用于存储此法分析器将生成的词法符号
	tokens := antlr.NewCommonTokenStream(lexer, antlr.LexerDefaultTokenChannel)

	// 新建一个语法分析器，处理词法符号缓冲区中的内容
	parser := NewConditionParser(tokens)
	parser.SetErrorHandler(antlr.NewDefaultErrorStrategy())
	parser.RemoveErrorListeners()
	parser.AddErrorListener(conError)

	tree := parser.Expr()
	var err error
	// parser error
	for _, v := range conError.Errors {
		if err == nil {
			err = v
			continue
		}
		err = errors.Wrap(err, v.Error())
	}

	if err != nil {
		return nil, err
	}
	return &Condition{tree: tree}, nil
}

func (c *Condition) Validate(data map[string]interface{}) (bool, error) {
	visitor := new(condition)
	visitor.data = data
	res := visitor.Visit(c.tree)

	if visitor.Err != nil {
		return false, visitor.Err
	}

	return res.(bool), nil
}

type condition struct {
	BaseConditionVisitor

	data map[string]interface{}
	Err  error
}

func (p *condition) Visit(tree antlr.ParseTree) interface{} {
	return tree.Accept(p)
}

func (p *condition) VisitGetNumExpr(ctx *GetNumExprContext) interface{} {
	keySource := ctx.ConditionKey().GetText()
	v := p.Visit(ctx.ConditionKey())
	if v == nil {
		return nil
	}
	colVal, err := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
	if err != nil {
		p.Err = errors.Wrap(ErrParseData, fmt.Sprintf("%v %v", keySource, err))
		return false
	}

	rightData := strings.Trim(ctx.Number().GetText(), `"`)
	data, err := strconv.ParseFloat(rightData, 64)
	if err != nil {
		p.Err = errors.Wrap(ErrParseData, fmt.Sprintf("%v %v", keySource, err))
		return false
	}
	code := ctx.GetOp().GetText()

	switch code {
	case ">":
		if colVal > data {
			return true
		}
	case "<":
		if colVal < data {
			return true
		}
	case "==":
		if colVal == data {
			return true
		}
	case "!=":
		if colVal != data {
			return true
		}
	case ">=":
		if colVal >= data {
			return true
		}
	case "<=":
		if colVal <= data {
			return true
		}
	}
	return false
}

func (p *condition) VisitParent(ctx *ParentContext) interface{} {
	return p.Visit(ctx.Expr())
}

func (p *condition) VisitGetStringExpr(ctx *GetStringExprContext) interface{} {
	v := p.Visit(ctx.ConditionKey())
	if v == nil {
		return nil
	}
	colVal := fmt.Sprintf("%v", v)

	data := strings.Trim(ctx.STRING().GetText(), `"`)

	code := ctx.GetOp().GetText()

	switch code {
	case "=~":
		if strings.Contains(colVal, data) {
			return true
		}
	case "!~":
		if !strings.Contains(colVal, data) {
			return true
		}
	case "==":
		if colVal == data {
			return true
		}
	case "!=":
		if colVal != data {
			return true
		}
	}
	return false
}

func (p *condition) VisitGetArrayExpr(ctx *GetArrayExprContext) interface{} {
	keySource := ctx.ConditionKey().GetText()
	colVal := p.Visit(ctx.ConditionKey())
	if colVal == nil {
		return false
	}

	var data []interface{}
	err := json.Unmarshal([]byte(ctx.Array().GetText()), &data)
	if err != nil {
		p.Err = errors.Wrap(ErrParseData, fmt.Sprintf("%v %v", keySource, err))
		return false
	}

	code := strings.ToLower(
		strings.ReplaceAll(ctx.GetOp().GetText(), " ", ""),
	)

	switch code {
	case "in":
		for _, v := range data {
			if fmt.Sprintf("%v", colVal) == fmt.Sprintf("%v", v) {
				return true
			}
		}
	case "notin":
		res := true
		for _, v := range data {
			if fmt.Sprintf("%v", colVal) == fmt.Sprintf("%v", v) {
				res = false
				break
			}
		}
		return res
	}

	return false
}

func (p *condition) VisitAndOr(ctx *AndOrContext) interface{} {
	expr0 := p.Visit(ctx.Expr(0))
	expr1 := p.Visit(ctx.Expr(1))
	if expr0 == nil {
		p.Err = fmt.Errorf("err,%s", ctx.Expr(0).GetText())
		return nil
	}

	if expr1 == nil {
		p.Err = fmt.Errorf("err,%s", ctx.Expr(1).GetText())
		return nil
	}

	op := strings.ToLower(ctx.GetOp().GetText())
	switch op {
	case "and":
		return expr0.(bool) && expr1.(bool)
	case "or":
		return expr0.(bool) || expr1.(bool)
	}
	return false
}

func (p *condition) VisitConditionKey(ctx *ConditionKeyContext) interface{} {
	txt := ctx.COL().GetText()
	v, ok := p.data[txt]
	if !ok {
		column := ctx.GetStart().GetColumn()
		p.Err = errors.Wrap(ErrNotFoundCol, fmt.Sprintf("column:%d, '%s' isn't exist", column, txt))
		return nil
	}
	return v
}

func (p *condition) VisitArray(ctx *ArrayContext) interface{} {
	return nil
}

func (p *condition) VisitNumber(ctx *NumberContext) interface{} {
	return nil
}
