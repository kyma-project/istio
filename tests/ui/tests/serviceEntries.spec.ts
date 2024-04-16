import {generateNamespaceName, generateRandomName} from "../support";

context('Service Entries', () => {

    let seName: string;
    let namespaceName: string;

    beforeEach(() => {
        seName = generateRandomName("test-se");
        namespaceName = generateNamespaceName();
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
    });

    afterEach(() => {
        cy.deleteNamespace(namespaceName);
    });

    it('should create new', () => {
        cy.navigateToServiceEntries(namespaceName);

        cy.clickCreateButton();

        cy.serviceEntryTypeName(seName);
        cy.serviceEntryTypeHost("test.com");
        cy.serviceEntrySelectResolution("STATIC");
        cy.serviceEntrySelectLocation("MESH_EXTERNAL");
        cy.serviceEntryTypeAddress("192.192.192.192/24");

        cy.clickCreateButton()

        cy.contains(seName).should('be.visible');
        cy.get('#content-wrap')
            .should('include.text', "STATIC")
            .and('include.text', "MESH_EXTERNAL")
            .and('include.text', "test.com")
            .and('include.text', "192.192.192.192/24");
    });

});
