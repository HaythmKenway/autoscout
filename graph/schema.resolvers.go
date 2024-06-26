package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.45

import (
	"context"
	"fmt"

	"github.com/HaythmKenway/autoscout/pkg/httpx"
	"github.com/HaythmKenway/autoscout/graph/model"
	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

// AddTarget is the resolver for the addTarget field.
func (r *mutationResolver) AddTarget(ctx context.Context, input model.TargetInput) (*model.Target, error) {
	status, err := db.AddTarget(input.Target)
	if err != nil {
		return nil, err
	}
	target := &model.Target{
		Target: input.Target,
		Status: status,
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
		Target: input.Target,
		Status: status,
	}
	return target, nil
}

// Targets is the resolver for the targets field.
func (r *queryResolver) Targets(ctx context.Context) ([]*model.Target, error) {
	targets, err := db.GetTargetsFromTable()
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
	subDomains, err := db.GetSubsFromTable(target)
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

// RunScan is the resolver for the runScan field.
func (r *queryResolver) RunScan(ctx context.Context, target string) ([]*model.Target, error) {
	db.SubdomainEnum(target)
	subDomains, err := db.GetSubsFromTable(target)
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

// GetData is the resolver for the getData field.
func (r *queryResolver) GetData(ctx context.Context, target string) ([]*model.Information, error) {
	data, err := db.GetDataFromTable(target)
	
	if(err!=nil && err.Error() == "target not found") {
		httpx.Httpx(target)
		data, err = db.GetDataFromTable(target)
		if err != nil {
			return nil, fmt.Errorf("No Data found")
		}
	}
	localUtils.CheckError(err)
	if err != nil {
		return nil, err
	}
	var result []*model.Information
	result = append(result, &model.Information{
		Title:      data[0],
		URL:        data[1],
		Host:       data[2],
		StatusCode: data[3],
		Scheme:     data[4],
		A:          data[5],
		Cname:      data[6],
		Tech:       data[7],
		IP:         data[8],
		Port:       data[9],
	})
	return result, err
}

// Mutation returns MutationResolver implementation.
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
