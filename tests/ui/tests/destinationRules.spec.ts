import {generateNamespaceName, generateRandomName} from "../support";

context('Test Destination Rules', () => {

    let drName: string;
    let namespaceName: string;

    beforeEach(() => {
        drName = generateRandomName("test-dr");
        namespaceName = generateNamespaceName();
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
    });

    afterEach(() => {
        cy.deleteNamespace(namespaceName);
    });

    it('should create new', () => {
        cy.navigateToDestinationRules(namespaceName);

        cy.clickCreateButton();

        cy.destinationRuleTypeName(drName);
        cy.destinationRuleTypeHost("ratings.prod.svc.cluster.local");
        cy.clickCreateButton();

        cy.contains(drName).should('be.visible');
        cy.contains("ratings.prod.svc.cluster.local").should('be.visible');
        cy.contains('Subsets').should('not.exist');
        cy.contains('Workload Selector').should('not.exist');
    });

});
