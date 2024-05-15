import {generateNamespaceName, generateRandomName} from "../support";
import {JwtRule} from "../support/resource/requestAuthentication";

context('RequestAuthentications', () => {

    let namespaceName: string;
    let reqAuthName: string;

    beforeEach(() => {
        namespaceName = generateNamespaceName();
        reqAuthName = generateRandomName("test-");
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
    });

    afterEach(() => {
        cy.deleteNamespace(namespaceName);
    });

    it('should create new with two JWT Rules and no selector', () => {
        cy.navigateToRequestAuthentications(namespaceName);
        cy.clickCreateButton();

        cy.requestAuthenticationTypeName(reqAuthName);

        const jwtRule: JwtRule = {
            issuer: "test-issuer@example.com",
            jwksUri: "https://test-jwksUri@example.com",
            audiences: ["test-audience1", "test-audience2"],
            fromParams: ["test-fromParams1", "test-fromParams2"],
            fromCookies: ["test-fromCookies1", "test-fromCookies2"],
            fromHeaders: [
                {
                    name: "test-header1",
                    prefix: "test-prefix1"
                },
                {
                    name: "test-header2",
                    prefix: "test-prefix2"
                }]
        }
        const jwtRule2: JwtRule = {
            issuer: "second-issuer@example.com",
            jwksUri: "https://second-jwksUri@example.com",
            audiences: ["second-test-audience1", "second-test-audience2"],
            fromParams: ["second-test-fromParams1", "second-test-fromParams2"],
            fromCookies: ["second-test-fromCookies1", "second-test-fromCookies2"],
            fromHeaders: [
                {
                    name: "second-test-header1",
                    prefix: "second-test-prefix1"
                },
                {
                    name: "second-test-header2",
                    prefix: "second-test-prefix2"
                }]
        }
        cy.requestAuthenticationAddJwtRules([jwtRule, jwtRule2]);

        cy.clickCreateButton();

        // Verify the details
        cy.contains(reqAuthName).should('be.visible');

        assertJwtRule(jwtRule);
        assertJwtRule(jwtRule2);

        cy.contains("Matches all Pods in the Namespace")
    });

    it('should render pod for selector in details', () => {

        const podName = generateRandomName("reqauth-pod")
        cy.createHttpbinSleepPod(podName, namespaceName);
        cy.navigateToRequestAuthentications(namespaceName);
        cy.clickCreateButton();

        cy.requestAuthenticationTypeName(reqAuthName);
        cy.requestAuthenticationAddMatchLabel({key: "app", value: podName});
        cy.clickCreateButton();

        // Verify the details
        cy.contains(`${podName}`).should('be.visible');
        cy.contains(`app=${podName}`).should('be.visible');
    });
})

function assertJwtRule(rule: JwtRule) {
    cy.contains(`Issuer ${rule.issuer}`).should('be.visible').click();
    cy.get('[data-testid="collapse-content"]').contains(rule.issuer).should('be.visible');
    cy.get('[data-testid="collapse-content"]').contains(rule.jwksUri).should('be.visible');

    rule.audiences.forEach((audience) => {
        cy.get('[data-testid="collapse-content"]').contains(audience).should('be.visible');
    });
    rule.fromParams.forEach((param) => {
        cy.get('[data-testid="collapse-content"]').contains(param).should('be.visible');
    });
    rule.fromCookies.forEach((cookie) => {
        cy.get('[data-testid="collapse-content"]').contains(cookie).should('be.visible');
    });
    rule.fromHeaders.forEach((header) => {
        cy.contains(`Header ${header.name}`).should('be.visible').click();
        cy.contains(header.prefix).should('be.visible');
    });

    cy.contains(`Issuer ${rule.issuer}`).should('be.visible').click();
}

