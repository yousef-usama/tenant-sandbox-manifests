#!/usr/bin/env python3
"""
Tenant Sandbox Multi-Tenant EKS Platform Architecture Diagram
Generates high-resolution Lucidchart-style visual architecture overview using Python Diagrams & Graphviz.
"""

import os
import sys

# Ensure script runs with working directory in docs/
DOCS_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "docs"))
os.chdir(DOCS_DIR)

from diagrams import Diagram, Cluster, Edge
from diagrams.aws.network import Route53, ALB, NATGateway
from diagrams.aws.security import ACM, SecretsManager, IAMRole
from diagrams.aws.compute import EKS, EC2, ECR
from diagrams.aws.storage import S3, EBS
from diagrams.onprem.client import Users
from diagrams.onprem.gitops import Argocd
from diagrams.onprem.monitoring import Prometheus, Grafana
from diagrams.onprem.database import PostgreSQL
from diagrams.onprem.vcs import Github
from diagrams.k8s.network import NetworkPolicy
from diagrams.k8s.compute import Pod
from diagrams.k8s.ecosystem import ExternalDns
from diagrams.custom import Custom

# Path to custom logos
ASSETS_DIR = os.path.join(DOCS_DIR, "diagram-assets")
FALCO_LOGO = os.path.join(ASSETS_DIR, "falco.png")
CERT_MANAGER_LOGO = os.path.join(ASSETS_DIR, "cert-manager.png")

graph_attr = {
    "fontsize": "28",
    "fontname": "Helvetica Neue, Arial, sans-serif",
    "fontcolor": "#0f172a",
    "bgcolor": "#ffffff",
    "pad": "1.0",
    "splines": "ortho",
    "nodesep": "0.8",
    "ranksep": "1.2",
}

node_attr = {
    "fontsize": "11",
    "fontname": "Helvetica Neue, Arial, sans-serif",
    "fontcolor": "#1e293b",
}

cluster_attr_vpc = {
    "fontsize": "16",
    "fontname": "Helvetica Neue, Arial, sans-serif",
    "fontcolor": "#0369a1",
    "bgcolor": "#f0f9ff",
    "style": "rounded,dashed",
    "penwidth": "2.0",
    "pencolor": "#38bdf8",
}

cluster_attr_subnets = {
    "fontsize": "14",
    "fontname": "Helvetica Neue, Arial, sans-serif",
    "fontcolor": "#0284c7",
    "bgcolor": "#e0f2fe",
    "style": "rounded",
    "pencolor": "#7dd3fc",
}

cluster_attr_eks = {
    "fontsize": "15",
    "fontname": "Helvetica Neue, Arial, sans-serif",
    "fontcolor": "#1e40af",
    "bgcolor": "#f8fafc",
    "style": "rounded",
    "penwidth": "1.8",
    "pencolor": "#60a5fa",
}

cluster_attr_tenant_a = {
    "fontsize": "12",
    "fontname": "Helvetica Neue, Arial, sans-serif",
    "fontcolor": "#4338ca",
    "bgcolor": "#eef2ff",
    "style": "rounded",
    "penwidth": "1.4",
    "pencolor": "#818cf8",
}

cluster_attr_tenant_b = {
    "fontsize": "12",
    "fontname": "Helvetica Neue, Arial, sans-serif",
    "fontcolor": "#065f46",
    "bgcolor": "#ecfdf5",
    "style": "rounded",
    "penwidth": "1.4",
    "pencolor": "#34d399",
}

cluster_attr_tenant_c = {
    "fontsize": "12",
    "fontname": "Helvetica Neue, Arial, sans-serif",
    "fontcolor": "#92400e",
    "bgcolor": "#fffbeb",
    "style": "rounded",
    "penwidth": "1.4",
    "pencolor": "#fbbf24",
}

cluster_attr_platform = {
    "fontsize": "13",
    "fontname": "Helvetica Neue, Arial, sans-serif",
    "fontcolor": "#334155",
    "bgcolor": "#f1f5f9",
    "style": "rounded",
    "penwidth": "1.4",
    "pencolor": "#94a3b8",
}

cluster_attr_aws_services = {
    "fontsize": "14",
    "fontname": "Helvetica Neue, Arial, sans-serif",
    "fontcolor": "#c2410c",
    "bgcolor": "#fff7ed",
    "style": "rounded,dashed",
    "penwidth": "1.5",
    "pencolor": "#fb923c",
}

with Diagram(
    name="Tenant Sandbox: Multi-Tenant Cloud Platform Architecture",
    filename="architecture-overview",
    show=False,
    direction="TB",
    graph_attr=graph_attr,
    node_attr=node_attr,
    outformat="png",
):
    # Top Row: Users, GitOps Source, Route 53, ACM
    with Cluster("Public Ingress & DNS Layer", graph_attr={"bgcolor": "#ffffff", "pencolor": "#e2e8f0", "style": "rounded"}):
        users = Users("End Users / Tenants\n(Web UI & API Clients)")
        dns = Route53("Amazon Route 53\n*.yousef-usama-cloud.click")
        acm = ACM("AWS ACM Certificate\n(Wildcard *.domain)")
        github = Github("GitHub Repositories\n(Infra & Manifests)")

    users >> Edge(label="HTTPS (443)", color="#0284c7", style="bold") >> dns

    with Cluster("AWS VPC (10.40.0.0/16) - Frankfurt (eu-central-1)", graph_attr=cluster_attr_vpc):
        with Cluster("Public Subnets (Multi-AZ)", graph_attr=cluster_attr_subnets):
            alb = ALB("Shared AWS Ingress ALB\n(Host & Path Routing)")
            nat_gateways = NATGateway("NAT EC2 Instances (3x t4g.micro)\n(AL2023 IP Forwarding)")

        dns >> Edge(color="#0284c7", style="bold") >> alb
        acm - Edge(color="#64748b", style="dashed", label="TLS Offloading") - alb

        with Cluster("Private Subnets - Amazon EKS Cluster v1.31 (Spot Worker Nodes)", graph_attr=cluster_attr_eks):
            # Tenant Namespaces Side-by-Side
            with Cluster("Namespace: tenant-a", graph_attr=cluster_attr_tenant_a):
                api_a = Pod("feedback-api\n(tenant-a)")
                db_a = PostgreSQL("PostgreSQL 16\n(tenant-a DB)")
                netpol_a = NetworkPolicy("NetworkPolicy\n(Default-Deny)")
                api_a >> Edge(color="#4f46e5", label="Port 5432") >> db_a

            with Cluster("Namespace: tenant-b", graph_attr=cluster_attr_tenant_b):
                api_b = Pod("feedback-api\n(tenant-b)")
                db_b = PostgreSQL("PostgreSQL 16\n(tenant-b DB)")
                netpol_b = NetworkPolicy("NetworkPolicy\n(Default-Deny)")
                api_b >> Edge(color="#059669", label="Port 5432") >> db_b

            with Cluster("Namespace: tenant-c", graph_attr=cluster_attr_tenant_c):
                api_c = Pod("feedback-api\n(tenant-c)")
                db_c = PostgreSQL("PostgreSQL 16\n(tenant-c DB)")
                netpol_c = NetworkPolicy("NetworkPolicy\n(Default-Deny)")
                api_c >> Edge(color="#d97706", label="Port 5432") >> db_c

            # Platform & Security GitOps Add-ons Cluster
            with Cluster("Platform & Security Stack (Sync Waves 0-4)", graph_attr=cluster_attr_platform):
                argo = Argocd("Argo CD Control Plane\n(App-of-Apps Root)")
                prom = Prometheus("Prometheus Engine\n(ServiceMonitors)")
                graf = Grafana("Grafana Dashboards\n(Multi-Tenant Metrics)")
                falco = Custom("Falco Runtime\n(eBPF Threat Detection)", FALCO_LOGO)
                edns = ExternalDns("ExternalDNS\n(Route 53 Sync)")
                eso = SecretsManager("External Secrets Operator\n(ClusterSecretStore)")
                cm = Custom("cert-manager\n(Internal PKI / Webhooks)", CERT_MANAGER_LOGO)

                prom >> Edge(color="#f97316") >> graf

        # AWS Cloud Managed Services Layer
        with Cluster("AWS Cloud Managed Services & Per-Tenant Data", graph_attr=cluster_attr_aws_services):
            ecr = ECR("Amazon ECR\n(feedback-api:tag)")
            aws_secrets = SecretsManager("AWS Secrets Manager\n(Tenant DB Passwords)")
            irsa = IAMRole("AWS IAM OIDC Provider\n(IRSA Token Exchange)")

            with Cluster("Tenant Isolated S3 Export Buckets", graph_attr={"bgcolor": "#ffedd5", "style": "rounded", "pencolor": "#fdba74"}):
                s3_a = S3("s3://tenant-sandbox\n-tenant-a-exports")
                s3_b = S3("s3://tenant-sandbox\n-tenant-b-exports")
                s3_c = S3("s3://tenant-sandbox\n-tenant-c-exports")

    # Connect Ingress to Tenants
    alb >> Edge(color="#6366f1", label="tenant-a.* / apex") >> api_a
    alb >> Edge(color="#10b981", label="tenant-b.*") >> api_b
    alb >> Edge(color="#f59e0b", label="tenant-c.*") >> api_c

    # GitOps Git Sync
    github >> Edge(color="#0f172a", style="dashed", label="Git Sync Waves 0-4") >> argo

    # Monitoring Scrape
    prom >> Edge(color="#f97316", style="dotted", label="Scrapes /metrics") >> api_a
    prom >> Edge(color="#f97316", style="dotted") >> api_b
    prom >> Edge(color="#f97316", style="dotted") >> api_c

    # External DNS to Route 53
    edns >> Edge(color="#2563eb", style="dashed", label="Upserts DNS Records") >> dns

    # ESO to Secrets Manager
    eso >> Edge(color="#9333ea", style="dashed", label="Syncs DB Secrets") >> aws_secrets

    # IRSA S3 Access
    api_a >> Edge(color="#4f46e5", style="dashed", label="IRSA (403 to B/C)") >> s3_a
    api_b >> Edge(color="#059669", style="dashed", label="IRSA (403 to A/C)") >> s3_b
    api_c >> Edge(color="#d97706", style="dashed", label="IRSA (403 to A/B)") >> s3_c

    # Container pulls from ECR
    ecr >> Edge(color="#475569", style="dotted", label="Multi-Arch Pull") >> api_a

    # Falco monitors tenant pods
    falco - Edge(color="#ef4444", style="dotted", label="eBPF Syscall Trace") - api_a
