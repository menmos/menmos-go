package payload

import "errors"

func loadTagNode(tag interface{}) (tagNode, error) {
	if tagStr, ok := tag.(string); ok {
		return tagNode{Tag: tagStr}, nil
	}
	return tagNode{}, errors.New("tag should be a string")
}

func loadKeyValueNode(data map[string]interface{}) (keyValueNode, error) {
	key, keyOk := data["key"]
	value, valueOk := data["value"]

	if !(keyOk && valueOk) {
		return keyValueNode{}, errors.New("invalid key/value condition")
	}

	if keyStr, keyStrOk := key.(string); keyStrOk {
		if valStr, valStrOk := value.(string); valStrOk {
			return keyValueNode{Key: keyStr, Value: valStr}, nil
		}
		return keyValueNode{}, errors.New("value is not a string")
	}

	return keyValueNode{}, errors.New("key is not a string")
}

func loadHasKeyNode(key interface{}) (hasKeyNode, error) {
	if keyStr, ok := key.(string); ok {
		return hasKeyNode{Key: keyStr}, nil
	}
	return hasKeyNode{}, errors.New("key should be a string")
}

func loadParentNode(parent interface{}) (parentNode, error) {
	if parentStr, ok := parent.(string); ok {
		return parentNode{Parent: parentStr}, nil
	}
	return parentNode{}, errors.New("parent should be a string")
}

func loadExpressionTuple(data interface{}) ([2]interface{}, error) {
	if pair, ok := data.([]interface{}); ok {
		if len(pair) != 2 {
			return [2]interface{}{nil, nil}, errors.New("and/or bodies must be arrays of length two")
		}

		lhs, err := loadExpressionBody(pair[0])
		if err != nil {
			return [2]interface{}{nil, nil}, err
		}

		rhs, err := loadExpressionBody(pair[1])
		if err != nil {
			return [2]interface{}{nil, nil}, err
		}

		return [2]interface{}{lhs, rhs}, nil
	}
	return [2]interface{}{nil, nil}, errors.New("and/or must be a pair of expressions")
}

func loadAndNode(data interface{}) (andNode, error) {
	pair, err := loadExpressionTuple(data)
	return andNode{And: pair}, err
}

func loadOrNode(data interface{}) (orNode, error) {
	pair, err := loadExpressionTuple(data)
	return orNode{Or: pair}, err
}

func loadExpressionBody(dataObject interface{}) (interface{}, error) {
	data, dataOk := dataObject.(map[string]interface{})
	if !dataOk {
		return nil, errors.New("expression should be an object")
	}

	if tag, ok := data["tag"]; ok {
		return loadTagNode(tag)
	} else if _, ok := data["value"]; ok {
		return loadKeyValueNode(data)
	} else if key, ok := data["key"]; ok {
		return loadHasKeyNode(key)
	} else if parent, ok := data["parent"]; ok {
		return loadParentNode(parent)
	} else if notSubExpression, ok := data["not"]; ok {
		expr, err := loadExpressionBody(notSubExpression)
		return notNode{Not: expr}, err
	} else if andData, ok := data["and"]; ok {
		return loadAndNode(andData)
	} else if orData, ok := data["or"]; ok {
		return loadOrNode(orData)
	}

	return nil, errors.New("unknown expression")
}
