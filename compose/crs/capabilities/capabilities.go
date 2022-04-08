package capabilities

type (
	Capability string
	Set        []Capability
)

const (
	Create  Capability = "create"
	Update             = "update"
	Search             = "search"
	Paging             = "paging"
	Stats              = "stats"
	Sorting            = "sorting"
	RBAC               = "RBAC"

	// @todo identify and define the rest
)

var (
	full = Set{
		Create,
		Update,
		Search,
		Paging,
		Stats,
		Sorting,
		RBAC,
	}

	createCapabilities = Set{
		RBAC,
		Create,
	}

	updateCapabilities = Set{
		RBAC,
		Update,
	}

	searchCapabilities = Set{
		Search,
		Paging,
		Stats,
		RBAC,
		Sorting,
	}
)

func FullCapabilities() (cc Set) {
	return full.Union(nil)
}

func CreateCapabilities(reqiested Set) (required Set) {
	required = Set{
		Create,
	}

	return required.Union(createCapabilities.Intersect(reqiested))
}

func UpdateCapabilities(reqiested Set) (required Set) {
	required = Set{
		Update,
	}

	return required.Union(updateCapabilities.Intersect(reqiested))
}

func SearchCapabilities(reqiested Set) (required Set) {
	required = Set{
		Search,
	}

	return required.Union(searchCapabilities.Intersect(reqiested))
}

func (available Set) Can(requested ...Capability) bool {
	if len(requested) > 0 && len(available) == 0 {
		return false
	}

	for _, r := range requested {
		for _, a := range available {
			if r == a {
				goto next
			}
		}
		return false

	next:
		continue
	}
	return true
}

func (aa Set) Intersect(bb Set) (cc Set) {
	for _, a := range aa {
		for _, b := range bb {
			if a == b {
				cc = append(cc, a)
				break
			}
		}
	}

	return
}

func (aa Set) Union(bb Set) (cc Set) {
	ix := make(map[Capability]bool)
	for _, c := range append(aa, bb...) {
		if !ix[c] {
			ix[c] = true
			cc = append(cc, c)
		}
	}
	return
}
