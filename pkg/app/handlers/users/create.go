package users

import (
	"net/http"

	cerr "github.com/aserto-dev/errors"
	dsc "github.com/aserto-dev/go-directory/aserto/directory/common/v3"
	dsw "github.com/aserto-dev/go-directory/aserto/directory/writer/v3"
	"github.com/aserto-dev/go-directory/pkg/derr"
	"github.com/aserto-dev/scim/pkg/common"
	"github.com/elimity-com/scim"
	serrors "github.com/elimity-com/scim/errors"
	"github.com/pkg/errors"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

func (u UsersResourceHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	u.logger.Trace().Any("attributes", attributes).Msg("creating user")
	object, err := common.ResourceAttributesToObject(attributes, "user", attributes["userName"].(string))
	if err != nil {
		return scim.Resource{}, serrors.ScimErrorInvalidSyntax
	}

	resp, err := u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
		Object: object,
	})
	if err != nil {
		if errors.Is(cerr.UnwrapAsertoError(err), derr.ErrAlreadyExists) {
			return scim.Resource{}, serrors.ScimErrorUniqueness
		}
		return scim.Resource{}, err
	}

	createdAt := resp.Result.CreatedAt.AsTime()
	updatedAt := resp.Result.UpdatedAt.AsTime()
	resource := common.ObjectToResource(resp.Result, scim.Meta{
		Created:      &createdAt,
		LastModified: &updatedAt,
		Version:      resp.Result.Etag,
	})

	if attributes["userName"] != nil {
		propsMap := make(map[string]interface{})
		propsMap["kind"] = "IDENTITY_KIND_USERNAME"
		props, err := structpb.NewStruct(propsMap)
		if err != nil {
			return scim.Resource{}, err
		}
		_, err = u.dirClient.Writer.SetObject(r.Context(), &dsw.SetObjectRequest{
			Object: &dsc.Object{
				Type:       "identity",
				Id:         attributes["userName"].(string),
				Properties: props,
			},
		})
		if err != nil {
			return scim.Resource{}, err
		}

		_, err = u.dirClient.Writer.SetRelation(r.Context(), &dsw.SetRelationRequest{
			Relation: &dsc.Relation{
				SubjectId:   resp.Result.Id,
				SubjectType: "user",
				Relation:    "identifier",
				ObjectType:  "identity",
				ObjectId:    attributes["userName"].(string),
			}})
		if err != nil {
			return scim.Resource{}, err
		}
	}

	if attributes["emails"] != nil && u.cfg.SCIM.CreateEmailIdentities {
		for _, m := range attributes["emails"].([]interface{}) {
			email := m.(map[string]interface{})
			if err != nil {
				return scim.Resource{}, err
			}
			if email["value"].(string) == attributes["userName"].(string) {
				continue
			}

			err = u.setIdentity(r.Context(), resp.Result.Id, email["value"].(string), "IDENTITY_KIND_EMAIL")
			if err != nil {
				return scim.Resource{}, err
			}
		}
	}

	if attributes["externalId"] != nil {
		externalID := attributes["externalId"]
		err = u.setIdentity(r.Context(), resp.Result.Id, externalID.(string), "IDENTITY_KIND_PID")
		if err != nil {
			return scim.Resource{}, err
		}
	}

	if attributes["groups"] != nil {
		err = u.setUserGroups(r.Context(), resp.Result.Id, attributes["groups"].([]string))
		if err != nil {
			return scim.Resource{}, err
		}
	}

	return resource, nil
}
