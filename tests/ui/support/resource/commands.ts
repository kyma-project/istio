import { AuthorizationPolicyCommands } from "./authorizationPolicy";
import { DestinationRuleCommands } from "./destinationRule";
import { GatewayCommands } from "./gateway";
import { VirtualServiceCommands } from "./virtualService";
import { ServiceEntryCommands } from "./serviceEntry";
import { SidecarCommands } from "./sidecar";
import { TelemetryCommands } from "./telemetries";
import {RequestAuthenticationCommands} from "./requestAuthentication";

export interface Commands extends AuthorizationPolicyCommands, TelemetryCommands, DestinationRuleCommands, GatewayCommands,
        VirtualServiceCommands, ServiceEntryCommands, SidecarCommands, RequestAuthenticationCommands {
}
