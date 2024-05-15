import { getK8sCurrentContext } from "./k8sclient";
import config from "./dashboard/config";
import Chainable = Cypress.Chainable;

export interface NavigationCommands {
    navigateToAuthorizationPolicy(name: string, namespace: string): void
    navigateToAuthorizationPolicies(namespace: string): void
    navigateToTelemetries(namespace: string): void
    navigateToTelemetry(name: string, namespace: string): void
    navigateToRequestAuthentications(namespace: string): void
    navigateToRequestAuthentication(name: string, namespace: string): void
    navigateToDestinationRule(name: string, namespace: string): void
    navigateToDestinationRules(namespace: string): void
    navigateToGateway(name: string, namespace: string): void
    navigateToGateways(namespace: string): void
    navigateToVirtualService(name: string, namespace: string): void
    navigateToVirtualServices(namespace: string): void
    navigateToServiceEntry(name: string, namespace: string): void
    navigateToServiceEntries(namespace: string): void
    navigateToSidecar(name: string, namespace: string): void
    navigateToSidecars(namespace: string): void
}

Cypress.Commands.add('navigateToAuthorizationPolicy', (namespace: string, name: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/authorizationpolicies/${name}`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToAuthorizationPolicies', (namespace: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/authorizationpolicies`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToTelemetries', (namespace: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/telemetries`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToTelemetry', (namespace: string, name: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/telemetries/${name}`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToDestinationRule', (namespace: string, name: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/destinationrules/${name}`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToDestinationRules', (namespace: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/destinationrules`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToGateway', (namespace: string, name: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/gateways/${name}`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToGateways', (namespace: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/gateways`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToVirtualService', (namespace: string, name: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/virtualservices/${name}`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToVirtualServices', (namespace: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/virtualservices`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToServiceEntry', (namespace: string, name: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/serviceentries/${name}`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToServiceEntries', (namespace: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/serviceentries`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToSidecar', (namespace: string, name: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/sidecars/${name}`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToSidecars', (namespace: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/sidecars`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToRequestAuthentication', (namespace: string, name: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/requestauthentications/${name}`)
        cy.wait(2000);
    });
});

Cypress.Commands.add('navigateToRequestAuthentications', (namespace: string): void => {
    cy.wrap(getK8sCurrentContext()).then((context) => {
        cy.visit(`${config.clusterAddress}/cluster/${context}/namespaces/${namespace}/requestauthentications`)
        cy.wait(2000);
    });
});
