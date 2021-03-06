package config

import (
	"fmt"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/mitchellh/mapstructure"
)

func (c *Config) processVaultAuths(list *ast.ObjectList, environment *Environment) error {
	if len(list.Items) == 0 {
		return nil
	}

	for _, authAST := range list.Items {
		x := authAST.Val.(*ast.ObjectType).List

		valid := []string{"config", "role", "type", "path", "max_lease_ttl", "default_lease_ttl"}
		if err := checkHCLKeys(x, valid); err != nil {
			return err
		}

		if len(authAST.Keys) != 1 {
			return fmt.Errorf("Missing auth name in line %+v", authAST.Keys[0].Pos())
		}

		authName := authAST.Keys[0].Token.Value().(string)

		typeAST := x.Filter("type")
		if len(typeAST.Items) != 1 {
			return fmt.Errorf("missing auth type in %s -> %s", environment.Name, authName)
		}

		authType := typeAST.Items[0].Val.(*ast.LiteralType).Token.Value().(string)

		mountMaxLeaseTTL := ""
		maxTTLAST := x.Filter("max_lease_ttl")
		if len(maxTTLAST.Items) == 1 {
			v := maxTTLAST.Items[0].Val.(*ast.LiteralType).Token.Value()
			switch t := v.(type) {
			default:
				return fmt.Errorf("unexpected type %T for %s -> %s -> max_lease_ttl", environment.Name, authName, t)
			case string:
				mountMaxLeaseTTL = v.(string)
			}
		} else if len(maxTTLAST.Items) > 1 {
			return fmt.Errorf("You can only specify max_lease_ttl once per mount in %s -> %s", environment.Name, authName)
		}

		mountDefaultLeaseTTL := ""
		defaultTTLAST := x.Filter("default_lease_ttl")
		if len(defaultTTLAST.Items) == 1 {
			v := defaultTTLAST.Items[0].Val.(*ast.LiteralType).Token.Value()
			switch t := v.(type) {
			default:
				return fmt.Errorf("unexpected type %T for %s -> %s -> default_lease_ttl", environment.Name, authName, t)
			case string:
				mountDefaultLeaseTTL = v.(string)
			}
		} else if len(defaultTTLAST.Items) > 1 {
			return fmt.Errorf("You can only specify default_lease_ttl once per mount in %s -> %s", environment.Name, authName)
		}

		auth := &Auth{
			Name:            authName,
			Type:            authType,
			Environment:     environment,
			DefaultLeaseTTL: mountDefaultLeaseTTL,
			MaxLeaseTTL:     mountMaxLeaseTTL,
		}

		configAST := x.Filter("config")
		if len(configAST.Items) > 0 {
			config, err := c.parseAuthConfig(configAST)
			if err != nil {
				return err
			}

			auth.Config = config
		}

		roleAST := x.Filter("role")
		if len(roleAST.Items) > 0 {
			roles, err := c.parseAuthRole(roleAST)
			if err != nil {
				return err
			}

			auth.Roles = roles
		}

		c.VaultAuths.Add(auth)
	}

	return nil
}

func (c *Config) parseAuthConfig(list *ast.ObjectList) ([]*AuthConfig, error) {
	configs := make([]*AuthConfig, 0)

	for _, authConfigAST := range list.Items {
		if len(authConfigAST.Keys) < 1 {
			return nil, fmt.Errorf("Missing auth role name in line %+v", authConfigAST.Keys[0].Pos())
		}

		var m map[string]interface{}
		if err := hcl.DecodeObject(&m, authConfigAST.Val); err != nil {
			return nil, err
		}

		var config AuthConfig
		config.Name = authConfigAST.Keys[0].Token.Value().(string)

		if err := mapstructure.WeakDecode(m, &config.Data); err != nil {
			return nil, err
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

func (c *Config) parseAuthRole(list *ast.ObjectList) ([]*AuthRole, error) {
	roles := make([]*AuthRole, 0)

	for _, config := range list.Items {
		if len(config.Keys) < 1 {
			return nil, fmt.Errorf("Missing auth role name in line %+v", config.Keys[0].Pos())
		}

		var m map[string]interface{}
		if err := hcl.DecodeObject(&m, config.Val); err != nil {
			return nil, err
		}

		var role AuthRole
		role.Name = config.Keys[0].Token.Value().(string)

		if err := mapstructure.WeakDecode(m, &role.Data); err != nil {
			return nil, err
		}

		roles = append(roles, &role)
	}

	return roles, nil
}
