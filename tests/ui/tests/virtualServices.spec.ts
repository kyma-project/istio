import 'cypress-file-upload';
import {generateNamespaceName, generateRandomName} from "../support";

context('Virtual Services', () => {

    let vsName: string;
    let namespaceName: string;

    beforeEach(() => {
        vsName = generateRandomName("test-vs");
        namespaceName = generateNamespaceName();
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
    });

    afterEach(() => {
        cy.deleteNamespace(namespaceName);
    });

    it('should create new', () => {
        cy.navigateToVirtualServices(namespaceName);

        cy.clickCreateButton();
        cy.virtualServiceTypeName(vsName);

        cy.virtualServiceAddHttpRoute({
            matchName: "test-match",
            uri: {
                prefix: 'prefix',
                value: "/wpcatalog",
            },
            redirect: {
                uri: '/v1/bookRatings',
                authority: 'newratings.default.svc.cluster.local',
            },
        });

        cy.clickCreateButton();

        cy.contains(vsName).should('be.visible');
        cy.get('[data-testid="collapse-button-close"]', {timeout: 10000}).click();

        cy.contains("test-match");
        cy.contains("prefix=/wpcatalog");
        cy.contains("/v1/bookRatings");
        cy.contains("newratings.default.svc.cluster.local");
    });

})
;
