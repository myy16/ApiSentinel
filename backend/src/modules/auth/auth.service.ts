import { db } from "../../lib/db.js";
import { users, organizations, memberships } from "../../db/schema/users.js";
import { eq } from "drizzle-orm";
import { hashPassword, comparePassword } from "../../lib/password.js";
import { generateAccessToken, generateRefreshToken, verifyToken } from "../../lib/jwt.js";
import { RegisterInput, LoginInput, UserRole } from "@apisentinel/shared";

export class AuthService {
  static async register(input: RegisterInput) {
    const existing = await db.query.users.findFirst({
      where: eq(users.email, input.email.toLowerCase()),
    });

    if (existing) {
      const error = new Error("Bu e-posta adresi zaten kullanımda");
      (error as any).statusCode = 409;
      (error as any).code = "EMAIL_EXISTS";
      throw error;
    }

    const passwordHash = await hashPassword(input.password);

    // Create User, Default Organization and OWNER Membership in transaction
    return await db.transaction(async (tx) => {
      const [newUser] = await tx
        .insert(users)
        .values({
          email: input.email.toLowerCase(),
          passwordHash,
        })
        .returning({ id: users.id, email: users.email, createdAt: users.createdAt });

      const orgName = input.organizationName || `${newUser.email.split("@")[0]}'s Org`;
      const [newOrg] = await tx
        .insert(organizations)
        .values({
          name: orgName,
        })
        .returning({ id: organizations.id, name: organizations.name, createdAt: organizations.createdAt });

      await tx.insert(memberships).values({
        organizationId: newOrg.id,
        userId: newUser.id,
        role: UserRole.OWNER,
      });

      const tokenPayload = {
        userId: newUser.id,
        email: newUser.email,
        organizationId: newOrg.id,
        role: UserRole.OWNER,
      };

      const accessToken = await generateAccessToken(tokenPayload);
      const refreshToken = await generateRefreshToken(tokenPayload);

      return {
        user: newUser,
        organization: newOrg,
        accessToken,
        refreshToken,
      };
    });
  }

  static async login(input: LoginInput) {
    const user = await db.query.users.findFirst({
      where: eq(users.email, input.email.toLowerCase()),
    });

    if (!user) {
      const error = new Error("E-posta veya şifre hatalı");
      (error as any).statusCode = 401;
      (error as any).code = "INVALID_CREDENTIALS";
      throw error;
    }

    const isValid = await comparePassword(input.password, user.passwordHash);
    if (!isValid) {
      const error = new Error("E-posta veya şifre hatalı");
      (error as any).statusCode = 401;
      (error as any).code = "INVALID_CREDENTIALS";
      throw error;
    }

    // Find user's primary organization
    const membership = await db.query.memberships.findFirst({
      where: eq(memberships.userId, user.id),
    });

    let organization = null;
    let role = UserRole.DEVELOPER;

    if (membership) {
      organization = await db.query.organizations.findFirst({
        where: eq(organizations.id, membership.organizationId),
      });
      role = membership.role as UserRole;
    }

    const tokenPayload = {
      userId: user.id,
      email: user.email,
      organizationId: organization?.id,
      role,
    };

    const accessToken = await generateAccessToken(tokenPayload);
    const refreshToken = await generateRefreshToken(tokenPayload);

    return {
      user: {
        id: user.id,
        email: user.email,
        createdAt: user.createdAt,
      },
      organization,
      accessToken,
      refreshToken,
    };
  }

  static async refresh(refreshToken: string) {
    const payload = await verifyToken(refreshToken, "refresh");

    const user = await db.query.users.findFirst({
      where: eq(users.id, payload.userId),
    });

    if (!user) {
      const error = new Error("Kullanıcı bulunamadı");
      (error as any).statusCode = 401;
      (error as any).code = "USER_NOT_FOUND";
      throw error;
    }

    const tokenPayload = {
      userId: user.id,
      email: user.email,
      organizationId: payload.organizationId,
      role: payload.role,
    };

    const newAccessToken = await generateAccessToken(tokenPayload);
    const newRefreshToken = await generateRefreshToken(tokenPayload);

    return {
      accessToken: newAccessToken,
      refreshToken: newRefreshToken,
    };
  }

  static async getMe(userId: string) {
    const user = await db.query.users.findFirst({
      where: eq(users.id, userId),
      columns: {
        id: true,
        email: true,
        createdAt: true,
      },
    });

    if (!user) {
      const error = new Error("Kullanıcı bulunamadı");
      (error as any).statusCode = 404;
      (error as any).code = "NOT_FOUND";
      throw error;
    }

    const userMemberships = await db.query.memberships.findMany({
      where: eq(memberships.userId, userId),
    });

    return {
      user,
      memberships: userMemberships,
    };
  }
}
