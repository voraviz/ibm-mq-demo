package com.example.config;

import jakarta.inject.Qualifier;

import java.lang.annotation.Retention;
import java.lang.annotation.Target;

import static java.lang.annotation.ElementType.FIELD;
import static java.lang.annotation.ElementType.METHOD;
import static java.lang.annotation.ElementType.PARAMETER;
import static java.lang.annotation.RetentionPolicy.RUNTIME;

/**
 * Qualifies the short-lived {@code ConnectionFactory} used only for one-shot
 * connectivity probes (e.g. {@code /api/info}). Unlike the default factory used
 * by the consumer/producer, this one has client auto-reconnect disabled so a
 * probe fails fast instead of inheriting reconnect semantics.
 */
@Qualifier
@Retention(RUNTIME)
@Target({FIELD, METHOD, PARAMETER})
public @interface ProbeConnection {
}
