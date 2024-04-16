import {generateNamespaceName, generateRandomName} from "../support";

context('Authorization Policies', () => {

    let apName: string;
    let namespaceName: string;

    beforeEach(() => {
        apName = generateRandomName("test-ap");
        namespaceName = generateNamespaceName();
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
    });

    afterEach(() => {
        cy.deleteNamespace(namespaceName);
    });

    it('should create new', () => {
        cy.navigateToAuthorizationPolicies(namespaceName);
        cy.clickCreateButton();

        cy.authorizationPolicySelectAction("AUDIT")
        cy.authorizationPolicyTypeName(apName);
        cy.authorizationPolicyAddRule({
            when: {
                key: 'request.auth.claims[iss]',
                value: 'https://test-value.com'
            },
            to: {
                operation: {
                    method: "GET",
                    path: '/user/profile/*'
                }
            }
        });


        cy.clickCreateButton();

        // Verify the details
        cy.contains(apName).should('be.visible');
        cy.contains("AUDIT").should('be.visible');
        cy.contains('Matches all Pods in the Namespace').should('be.visible');
        cy.contains('Rule #1 to when', { timeout: 10000 }).click();
        cy.contains('To #1 methods paths', { timeout: 10000 }).click();
        cy.contains("/user/profile/*").should('be.visible');
        cy.contains("request.auth.claims[iss]").should('be.visible');
        cy.contains("https://test-value.com").should('be.visible');
        cy.contains('Operation').should('be.visible');
        cy.contains("GET").should('be.visible');
    });


    it('should update action', () => {
        cy.createAuthorizationPolicy(namespaceName, apName);
        cy.navigateToAuthorizationPolicy(namespaceName, apName);

        cy.clickEditTab();
        cy.authorizationPolicySelectAction("ALLOW")
        cy.clickSaveButton();
        cy.clickViewTab();

        cy.contains("ALLOW");
    });

});
