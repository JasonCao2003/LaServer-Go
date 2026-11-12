package service

// The legacy tutorial acknowledges this reward without a server-side grant.
// Its client retries indefinitely unless the response is HTTP 200 with result=ok.
func (a *Service) RedeemReward(_ Request, _ Object) (Object, error) {
	return Object{"result": "ok"}, nil
}

func (a *Service) AckForgeRateReset(_ Request, _ Object) (Object, error) {
	return Object{"result": "ok", "update_forge_rates": []Object{{"source_type": "wax", "current_tier": 26, "current_level": 4, "tier_completion": 0.0, "level_completion": 0.0, "level_tier_index": 0, "level_tier_count": 5, "reset_acknowledged": true, "enabled": true}}}, nil
}

// The old update endpoint only echoed the gifter; it did not modify an order.
func (a *Service) CommerceUpdate(_ Request, m Object) (Object, error) {
	gifter := text(m, "gifter")
	if gifter == "" {
		gifter = text(m, "user")
	}
	return Object{"errorcode": 0, "result": "ok", "status": "ok", "gifter": gifter}, nil
}

func (a *Service) CommerceProductInfo(_ Request, _ Object) (Object, error) {
	var body Object
	if err := legacyResponse("account__commerce__product_info", &body); err != nil {
		return Object{"products": []Object{}, "iaplist": []Object{}}, nil
	}
	return body, nil
}
