package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const (
	TPRTenantsKind = "Tenant"
	TPRTenantName  = "tenants"
)

type TenantsGetter interface {
	Tenants(namespace string) TenantInterface
}

type TenantInterface interface {
	Create(*Tenant) (*Tenant, error)
	Get(name string) (*Tenant, error)
	Update(*Tenant) (*Tenant, error)
	Delete(name string, options *metav1.DeleteOptions) error
	List(opts metav1.ListOptions) (runtime.Object, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
}

type tenants struct {
	restClient rest.Interface
	client     *dynamic.ResourceClient
	ns         string
}

func newTenants(r rest.Interface, c *dynamic.Client, namespace string) *tenants {
	return &tenants{
		r,
		c.Resource(
			&metav1.APIResource{
				Kind:       TPRTenantsKind,
				Name:       TPRTenantName,
				Namespaced: true,
			},
			namespace,
		),
		namespace,
	}
}

func (p *tenants) Create(o *Tenant) (*Tenant, error) {
	up, err := UnstructuredFromTenant(o)
	if err != nil {
		return nil, err
	}

	up, err = p.client.Create(up)
	if err != nil {
		return nil, err
	}

	return TenantFromUnstructured(up)
}

func (p *tenants) Get(name string) (*Tenant, error) {
	obj, err := p.client.Get(name)
	if err != nil {
		return nil, err
	}
	return TenantFromUnstructured(obj)
}

func (p *tenants) Update(o *Tenant) (*Tenant, error) {
	up, err := UnstructuredFromTenant(o)
	if err != nil {
		return nil, err
	}

	up, err = p.client.Update(up)
	if err != nil {
		return nil, err
	}

	return TenantFromUnstructured(up)
}

func (p *tenants) Delete(name string, options *metav1.DeleteOptions) error {
	return p.client.Delete(name, options)
}

func (p *tenants) List(opts metav1.ListOptions) (runtime.Object, error) {
	req := p.restClient.Get().
		Namespace(p.ns).
		Resource("tenants").
		// VersionedParams(&options, v1.ParameterCodec)
		FieldsSelectorParam(nil)

	b, err := req.DoRaw()
	if err != nil {
		return nil, err
	}
	var tena TenantList
	return &tena, json.Unmarshal(b, &tena)
}

func (p *tenants) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	r, err := p.restClient.Get().
		Prefix("watch").
		Namespace(p.ns).
		Resource("tenants").
		// VersionedParams(&options, v1.ParameterCodec).
		FieldsSelectorParam(nil).
		Stream()
	if err != nil {
		return nil, err
	}
	return watch.NewStreamWatcher(&tenantDecoder{
		dec:   json.NewDecoder(r),
		close: r.Close,
	}), nil
}

// TenantFromUnstructured unmarshals a Tenant object from dynamic client's unstructured
func TenantFromUnstructured(r *unstructured.Unstructured) (*Tenant, error) {
	b, err := json.Marshal(r.Object)
	if err != nil {
		return nil, err
	}
	var p Tenant
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	p.TypeMeta.Kind = TPRTenantsKind
	p.TypeMeta.APIVersion = TPRGroup + "/" + TPRVersion
	return &p, nil
}

// UnstructuredFromTenant marshals a Tenant object into dynamic client's unstructured
func UnstructuredFromTenant(p *Tenant) (*unstructured.Unstructured, error) {
	p.TypeMeta.Kind = TPRTenantsKind
	p.TypeMeta.APIVersion = TPRGroup + "/" + TPRVersion
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var r unstructured.Unstructured
	if err := json.Unmarshal(b, &r.Object); err != nil {
		return nil, err
	}
	return &r, nil
}

type tenantDecoder struct {
	dec   *json.Decoder
	close func() error
}

func (d *tenantDecoder) Close() {
	d.close()
}

func (d *tenantDecoder) Decode() (action watch.EventType, object runtime.Object, err error) {
	var e struct {
		Type   watch.EventType
		Object Tenant
	}
	if err := d.dec.Decode(&e); err != nil {
		return watch.Error, nil, err
	}
	return e.Type, &e.Object, nil
}
