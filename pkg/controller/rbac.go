package controller

import (
	kutilcore "github.com/appscode/kutil/core/v1"
	kutilrbac "github.com/appscode/kutil/rbac/v1beta1"
	// "github.com/kubedb/apimachinery/apis/kubedb"
	api "github.com/kubedb/apimachinery/apis/kubedb/v1alpha1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1beta1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) deleteRole(elasticsearch *api.Elasticsearch) error {
	// Delete existing Roles
	if err := c.Client.RbacV1beta1().Roles(elasticsearch.Namespace).Delete(elasticsearch.OffshootName(), nil); err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *Controller) ensureRole(elasticsearch *api.Elasticsearch) error {
	// Create new Roles
	_, _, err := kutilrbac.CreateOrPatchRole(
		c.Client,
		metav1.ObjectMeta{
			Name:      elasticsearch.OffshootName(),
			Namespace: elasticsearch.Namespace,
		},
		func(in *rbac.Role) *rbac.Role {
			in.Rules = []rbac.PolicyRule{
				{
					APIGroups: []string{core.GroupName},
					Resources: []string{"nodes"},
					Verbs:     []string{"list"},
				},
			}
			return in
		},
	)
	return err
}

func (c *Controller) deleteServiceAccount(elasticsearch *api.Elasticsearch) error {
	// Delete existing ServiceAccount
	if err := c.Client.CoreV1().ServiceAccounts(elasticsearch.Namespace).Delete(elasticsearch.OffshootName(), nil); err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *Controller) ensureServiceAccount(elasticsearch *api.Elasticsearch) error {
	// Create new ServiceAccount
	_, _, err := kutilcore.CreateOrPatchServiceAccount(
		c.Client,
		metav1.ObjectMeta{
			Name:      elasticsearch.OffshootName(),
			Namespace: elasticsearch.Namespace,
		},
		func(in *core.ServiceAccount) *core.ServiceAccount {
			return in
		},
	)
	return err
}

func (c *Controller) deleteRoleBinding(elasticsearch *api.Elasticsearch) error {
	// Delete existing RoleBindings
	if err := c.Client.RbacV1beta1().RoleBindings(elasticsearch.Namespace).Delete(elasticsearch.OffshootName(), nil); err != nil {
		if !kerr.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *Controller) ensureRoleBinding(elasticsearch *api.Elasticsearch) error {
	// Ensure new RoleBindings
	_, _, err := kutilrbac.CreateOrPatchRoleBinding(
		c.Client,
		metav1.ObjectMeta{
			Name:      elasticsearch.OffshootName(),
			Namespace: elasticsearch.Namespace,
		},
		func(in *rbac.RoleBinding) *rbac.RoleBinding {
			in.RoleRef = rbac.RoleRef{
				APIGroup: rbac.GroupName,
				Kind:     "Role",
				Name:     elasticsearch.OffshootName(),
			}
			in.Subjects = []rbac.Subject{
				{
					Kind:      rbac.ServiceAccountKind,
					Name:      elasticsearch.OffshootName(),
					Namespace: elasticsearch.Namespace,
				},
			}
			return in
		},
	)
	return err
}

func (c *Controller) ensureRBACStuff(elasticsearch *api.Elasticsearch) error {
	// Create New Role
	if err := c.ensureRole(elasticsearch); err != nil {
		return err
	}

	// Create New ServiceAccount
	if err := c.ensureServiceAccount(elasticsearch); err != nil {
		return err
	}

	// Create New RoleBinding
	if err := c.ensureRoleBinding(elasticsearch); err != nil {
		return err
	}

	return nil
}

func (c *Controller) deleteRBACStuff(elasticsearch *api.Elasticsearch) error {
	// Delete Existing Role
	if err := c.deleteRole(elasticsearch); err != nil {
		return err
	}

	// Delete ServiceAccount
	if err := c.deleteServiceAccount(elasticsearch); err != nil {
		return err
	}

	// Delete New RoleBinding
	if err := c.deleteRoleBinding(elasticsearch); err != nil {
		return err
	}
	return nil
}
