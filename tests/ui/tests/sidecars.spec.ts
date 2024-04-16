import {generateNamespaceName, generateRandomName} from "../support";

context('Sidecars', () => {

    let scName: string;
    let namespaceName: string;

    beforeEach(() => {
        scName = generateRandomName("test-sc");
        namespaceName = generateNamespaceName();
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
    });

    afterEach(() => {
        cy.deleteNamespace(namespaceName);
    });

    it('should create new sidecar with ingress', () => {
        cy.navigateToSidecars(namespaceName);
        cy.clickCreateButton();

        cy.sidecarTypeName(scName);
        cy.sidecarAddIngress({
            name: `ing-name-${scName}`,
            port: 80,
            protocol: "HTTP",
            defaultEndpoint: "127.0.0.1:8080"
        })

        cy.clickCreateButton();

        cy.contains(scName).should('be.visible');
        cy.contains(80);
        cy.contains(`ing-name-${scName}`);
        cy.contains("HTTP");
        cy.contains("127.0.0.1:8080");
    });
});
