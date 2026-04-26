package platformclient

import (
	"context"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	rulev1 "code-code.internal/go-contract/platform/rule/v1"
	skillv1 "code-code.internal/go-contract/platform/skill/v1"
)

func (s *Skills) List(ctx context.Context) ([]*managementv1.SkillListItem, error) {
	client, err := s.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.ListSkills(ctx, &managementv1.ListSkillsRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

func (s *Skills) Get(ctx context.Context, skillID string) (*skillv1.Skill, error) {
	client, err := s.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.GetSkill(ctx, &managementv1.GetSkillRequest{SkillId: skillID})
	if err != nil {
		return nil, err
	}
	return response.GetSkill(), nil
}

func (s *Skills) Create(ctx context.Context, request *managementv1.UpsertSkillRequest) (*skillv1.Skill, error) {
	client, err := s.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.CreateSkill(ctx, &managementv1.CreateSkillRequest{Skill: request})
	if err != nil {
		return nil, err
	}
	return response.GetSkill(), nil
}

func (s *Skills) Update(ctx context.Context, skillID string, request *managementv1.UpsertSkillRequest) (*skillv1.Skill, error) {
	client, err := s.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.UpdateSkill(ctx, &managementv1.UpdateSkillRequest{SkillId: skillID, Skill: request})
	if err != nil {
		return nil, err
	}
	return response.GetSkill(), nil
}

func (s *Skills) Delete(ctx context.Context, skillID string) error {
	client, err := s.client.requireProfile()
	if err != nil {
		return err
	}
	_, err = client.DeleteSkill(ctx, &managementv1.DeleteSkillRequest{SkillId: skillID})
	return err
}

func (r *Rules) List(ctx context.Context) ([]*managementv1.RuleListItem, error) {
	client, err := r.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.ListRules(ctx, &managementv1.ListRulesRequest{})
	if err != nil {
		return nil, err
	}
	return response.GetItems(), nil
}

func (r *Rules) Get(ctx context.Context, ruleID string) (*rulev1.Rule, error) {
	client, err := r.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.GetRule(ctx, &managementv1.GetRuleRequest{RuleId: ruleID})
	if err != nil {
		return nil, err
	}
	return response.GetRule(), nil
}

func (r *Rules) Create(ctx context.Context, request *managementv1.UpsertRuleRequest) (*rulev1.Rule, error) {
	client, err := r.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.CreateRule(ctx, &managementv1.CreateRuleRequest{Rule: request})
	if err != nil {
		return nil, err
	}
	return response.GetRule(), nil
}

func (r *Rules) Update(ctx context.Context, ruleID string, request *managementv1.UpsertRuleRequest) (*rulev1.Rule, error) {
	client, err := r.client.requireProfile()
	if err != nil {
		return nil, err
	}
	response, err := client.UpdateRule(ctx, &managementv1.UpdateRuleRequest{RuleId: ruleID, Rule: request})
	if err != nil {
		return nil, err
	}
	return response.GetRule(), nil
}

func (r *Rules) Delete(ctx context.Context, ruleID string) error {
	client, err := r.client.requireProfile()
	if err != nil {
		return err
	}
	_, err = client.DeleteRule(ctx, &managementv1.DeleteRuleRequest{RuleId: ruleID})
	return err
}
