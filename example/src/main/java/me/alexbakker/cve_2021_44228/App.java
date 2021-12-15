package me.alexbakker.cve_2021_44228;

import java.util.logging.Level;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;

public class App  {
    private static Logger log = LogManager.getLogger();

    public static void main(String[] args) {
        System.out.print("Enter a JNDI URI: ");
        String uri = System.console().readLine();
        log.error(uri);
    }
}
