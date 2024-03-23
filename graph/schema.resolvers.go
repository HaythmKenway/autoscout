package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.45

import (
	"context"

	"github.com/HaythmKenway/autoscout/graph/model"
	"github.com/HaythmKenway/autoscout/internal/db"
)

// AddTarget is the resolver for the addTarget field.
func (r *mutationResolver) AddTarget(ctx context.Context, input model.TargetInput) (*model.Target, error) {
	status, err := db.AddTarget(input.Target)
	if err != nil {
		return nil, err
	}
	target := &model.Target{
		Target: status,
	}
	return target, nil
}

// RemoveTarget is the resolver for the removeTarget field.
func (r *mutationResolver) RemoveTarget(ctx context.Context, input model.TargetInput) (*model.Target, error) {
	status, err := db.RemoveTarget(input.Target)
	if err != nil {
		return nil, err
	}
	target := &model.Target{
		Target: status,
	}
	return target, nil
}

// Targets is the resolver for the targets field.
func (r *queryResolver) Targets(ctx context.Context) ([]*model.Target, error) {
	targets, err := db.GetDomains()
	if err != nil {
		return nil, err
	}
	var result []*model.Target
	for _, target := range targets {
		result = append(result, &model.Target{
			Target: target,
		})
	}
	return result, nil
}

// SubDomain is the resolver for the subDomain field.
func (r *queryResolver) SubDomain(ctx context.Context, target string) ([]*model.Target, error) {
	subDomains, err := db.GetSubs(target)
	if err != nil {
		return nil, err
	}
	var result []*model.Target
	for _, subDomain := range subDomains {
		result = append(result, &model.Target{
			Target: subDomain,
		})
	}
	return result, nil
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
