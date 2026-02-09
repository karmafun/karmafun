import cert_manager.v1 as cert

# cSpell: words cloudforge openance letsencrypt emberstack

schema Domain:
    name: str
    dns_provider: str
    namespaces?: [str]

# Retrieve the values
_items = option("items")
_domains : [Domain] = [item.data.domains for item in _items if item.metadata.name == "autocloud-values"][0]

# Iterate on domain names and generate the Certificate resources
# TODO: We should have an extended schema for this
[ cert.Certificate {
    metadata.name = "all-" + domain.name.replace(".", "-")
    # The output filename for the generated
    metadata.annotations = {
        'config.karmafun.dev/path' = metadata.name + ".yaml"
    }
    spec.dnsNames = [ domain.name, "*." + domain.name ]
    spec.duration = "2160h0m0s"
    spec.renewBefore = "720h0m0s"
    spec.issuerRef.kind = "ClusterIssuer"
    spec.issuerRef.name = "letsencrypt"
    spec.secretName = metadata.name + "-tls"
    spec.secretTemplate.annotations = {
        'reflector.v1.k8s.emberstack.com/reflection-allowed' = "true"
        'reflector.v1.k8s.emberstack.com/reflection-auto-enabled' = "true"
        'reflector.v1.k8s.emberstack.com/reflection-allowed-namespaces' = ""
        'reflector.v1.k8s.emberstack.com/reflection-auto-namespaces' = ",".join(domain.namespaces) if domain.namespaces else ""
    }
} for domain in _domains ]
