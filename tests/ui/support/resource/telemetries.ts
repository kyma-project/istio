export interface TelemetryCommands {
        telemetryTypeName(name: string): void
        telemetryAddAccessLogging(accessLog: AccessLog): void
        telemetryAddTracing(tracing: Tracing): void
}

export type Tracing = {
        randomSamplingPercentage: number;
        provider: TracingProvider;
}

export type TracingProvider = {
        name: string;
}

export type AccessLog = {
        mode: "CLIENT_AND_SERVER" | "CLIENT" | "SERVER";
        filterExpression: string;
}

Cypress.Commands.add('telemetryTypeName', (name: string): void => {
        cy.inputClearAndType('ui5-input[aria-label="Telemetry name"]', name);
});

Cypress.Commands.add('telemetryAddAccessLogging', (accessLog: AccessLog): void => {
        cy.addFormGroupItem('[aria-label="expand Access logging configuration"]:visible');
        cy.chooseComboboxOption('[data-testid="spec.accessLogging.0.match.mode"]', accessLog.mode);
        cy.inputClearAndType('ui5-input[data-testid="spec.accessLogging.0.filter.expression"]', accessLog.filterExpression);
});

Cypress.Commands.add('telemetryAddTracing', (tracing: Tracing): void => {
        cy.addFormGroupItem('[aria-label="expand Tracing configuration"]:visible');
        cy.inputClearAndType('ui5-input[data-testid="spec.tracing.0.randomSamplingPercentage"]', tracing.randomSamplingPercentage.toString());

        cy.addFormGroupItem('[aria-label="expand Providers"]:visible');
        cy.inputClearAndType('ui5-input[data-testid="spec.tracing.0.providers.0.name"]', tracing.provider.name);
});
