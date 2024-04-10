import {generateNamespaceName, generateRandomName} from "../support";

context('Gateways', () => {

    let gwName: string;
    let namespaceName: string;

    beforeEach(() => {
        gwName = generateRandomName("test-ap");
        namespaceName = generateNamespaceName();
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
    });

    afterEach(() => {
        cy.deleteNamespace(namespaceName);
    });

    it('should create new', () => {
        cy.navigateToGateways(namespaceName);

        cy.clickCreateButton();
        cy.gatewayTypeName(gwName);
        cy.gatewayAddServer({
            port: 80,
            protocol: "HTTP",
            name: `${gwName}-server`,
            host: "*.example.com",
        });

        cy.clickCreateButton();

        cy.contains(gwName);
        cy.contains('istio=ingressgateway');
        cy.contains(`${gwName}-server`);
        cy.contains(80);
        cy.contains('*.example.com');
    });

});
