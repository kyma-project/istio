import { generateNamespaceName, generateRandomName } from "../support";

context('Telemetries', () => {

        let namespaceName: string;
        let telemetryName: string;

        beforeEach(() => {
                namespaceName = generateNamespaceName();
                telemetryName = generateRandomName("test-telemetry");
                cy.loginAndSelectCluster();
                cy.createNamespace(namespaceName);
        });

        afterEach(() => {
                cy.deleteNamespace(namespaceName);
        });

        it('should create new', () => {
                cy.navigateToTelemetries(namespaceName);
                cy.clickCreateButton();

                cy.telemetryTypeName(telemetryName);
                cy.telemetryAddAccessLogging({
                        mode: "SERVER",
                        filterExpression: "response.code >= 400"
                });

                cy.telemetryAddTracing({
                        randomSamplingPercentage: 100,
                        provider: {
                                name: "test-provider"
                        }
                });

                cy.clickCreateButton();

                // Verify the details
                cy.contains(telemetryName).should('be.visible');

                cy.contains("AccessLogging #1").should('be.visible').click();
                cy.contains("Tracing #1").should('be.visible').click();

                cy.contains("SERVER").should('be.visible');
                cy.contains("response.code >= 400").should('be.visible');
                cy.contains("100").should('be.visible');
                cy.contains("test-provider").should('be.visible');

        });
});

